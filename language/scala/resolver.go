package scala

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/index"
)

const (
	// ResolverImpLangPrivateKey stores the implementation language override.
	ResolverImpLangPrivateKey = "_resolve_imp_lang"
	debug                     = false
)

type importOrigin struct {
	Kind       string
	SourceFile *index.ScalaFileSpec
	Parent     string
	Children   []string // transitive imports triggered for an import
	Actual     string   // the effective string for the import.
}

func importMapKeys(in map[string]*importOrigin) []string {
	keys := make([]string, len(in))
	i := 0
	for k := range in {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func (io *importOrigin) String() string {
	var s string
	switch io.Kind {
	case "direct":
		s = io.Kind + " from " + filepath.Base(io.SourceFile.Filename)
		if io.Parent != "" {
			s += " (materialized from " + io.Parent + ")"
		}
	case "indirect":
		s = io.Kind + " from " + io.Parent
	case "superclass":
		s = io.Kind + " of " + io.Parent
	case "transitive":
		s = io.Kind + " via " + io.Parent
	default:
		return io.Kind
	}
	if len(io.Children) > 0 {
		s += fmt.Sprintf(" (requires %v)", io.Children)
	}
	return s
}

type labelImportMap map[label.Label]map[string]*importOrigin

func (m labelImportMap) Set(from label.Label, imp string, origin *importOrigin) {
	if all, ok := m[from]; ok {
		all[imp] = origin
	} else {
		m[from] = map[string]*importOrigin{imp: origin}
	}
	if debug {
		log.Printf(" --> resolved %q (%s) to %v", imp, origin.String(), from)
	}
}

func (m labelImportMap) String() string {
	var sb strings.Builder
	for from, imports := range m {
		sb.WriteString(from.String())
		sb.WriteString(":\n")
		for imp, origin := range imports {
			sb.WriteString(" -- ")
			sb.WriteString(imp)
			sb.WriteString(" -> ")
			sb.WriteString(origin.String())
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func gatherIndirectDependencies(c *config.Config, imports map[string]*importOrigin) {
	sc := getScalaConfig(c)

	stack := make(importStack, 0, len(imports))
	for k := range imports {
		stack = stack.push(k)
	}
	var imp string
	for !stack.empty() {
		stack, imp = stack.pop()
		for _, dep := range sc.GetIndirectDependencies(ScalaLangName, imp) {
			// make this is feature tooggle? for transitive indirects?
			stack = stack.push(dep)
			if _, ok := imports[dep]; !ok {
				imports[dep] = &importOrigin{Kind: "indirect", Parent: imp}
			}
		}
	}
}

func resolveTransitive(c *config.Config, ix *resolve.RuleIndex, importRegistry ScalaImportRegistry, impLang, kind string, from label.Label, imports map[string]*importOrigin, resolved labelImportMap) {
	// at this point we expect 'resolved' to hold a set of 'actual' imports that
	// represent concrete types. build a second set of imports to be resolved
	// from that set.
	transitiveImports := make(map[string]*importOrigin)

	for lbl, imps := range resolved {
		if lbl == label.NoLabel {
			continue
		}
		for _, origin := range imps {
			if origin.Actual == "" {
				log.Panicln("unknown actual import!", lbl, origin.String())
			}
			transitive, unresolved := importRegistry.TransitiveImports([]string{origin.Actual})
			if debug {
				if len(unresolved) > 0 {
					log.Println("unresolved transitive import:", unresolved, origin.Actual)
				}
			}
			for _, tImp := range transitive {
				// log.Println("transitive import:", imp, tImp)
				if _, ok := imports[tImp]; !ok {
					transitiveImports[tImp] = &importOrigin{Kind: "transitive", Parent: origin.Actual}
				}
			}
			// log.Println(from, "transitive imports:", origin.Actual, transitive)
			origin.Children = transitive
		}
	}

	// another round of indirects
	gatherIndirectDependencies(c, transitiveImports)

	// finally, resolve the transitive set
	resolveImports(c, ix, importRegistry, impLang, kind, from, transitiveImports, resolved)
}

func resolveImports(c *config.Config, ix *resolve.RuleIndex, importRegistry ScalaImportRegistry, impLang, kind string, from label.Label, imports map[string]*importOrigin, resolved labelImportMap) {
	sc := getScalaConfig(c)

	dbg := false
	for imp, origin := range imports {
		if dbg {
			log.Println("---", from, imp, "---")
			// log.Println("resolved:\n", resolved.String())
		}

		labels := resolveImport(c, ix, importRegistry, origin, impLang, imp, from, resolved)

		if len(labels) == 0 {
			resolved[label.NoLabel][imp] = origin
			if dbg {
				log.Println("unresolved:", imp)
			}
			continue
		}

		if len(labels) > 1 {
			original := labels
			disambiguated, err := importRegistry.Disambiguate(c, ix, resolve.ImportSpec{Lang: ScalaLangName, Imp: imp}, ScalaLangName, from, labels)
			if err != nil {
				log.Panicf("disambigation error: %v", err)
			}
			if dbg {
				log.Println(from, imp, original, "--[Disambiguate]-->", disambiguated)
			}
			labels = disambiguated

			for _, dep := range disambiguated {
				if dep == label.NoLabel || dep == PlatformLabel || from.Equal(dep) || isSameImport(sc, kind, from, dep) {
					continue
				}
				resolved.Set(dep, imp, origin)
			}
		} else {
			for _, dep := range labels {
				if dep == label.NoLabel || dep == PlatformLabel || from.Equal(dep) || isSameImport(sc, kind, from, dep) {
					continue
				}
				resolved.Set(dep, imp, origin)
			}
		}
	}
}

// resolveImport should return a different thing than
func resolveImport(c *config.Config, ix *resolve.RuleIndex, registry ScalaImportRegistry, origin *importOrigin, lang string, imp string, from label.Label, resolved labelImportMap) []label.Label {
	if debug {
		log.Println("resolveImport:", imp, origin.String())
	}

	// if the import is empty, we may have reached the root symbol.
	if imp == "" {
		return nil
	}

	labels := resolveAnyKind(c, ix, lang, imp, from)
	if debug {
		log.Println("resolveAnyKind:", imp, labels)
	}

	if len(labels) > 0 {
		origin.Actual = imp
		return dedupLabels(labels)
	}

	// if this is a _root_ import, try without
	if strings.HasPrefix(imp, "_root_.") {
		return resolveImport(c, ix, registry, origin, lang, strings.TrimPrefix(imp, "_root_."), from, resolved)
	}

	// if this is a wildcard import, try without
	if strings.HasSuffix(imp, "._") {
		return resolveImport(c, ix, registry, origin, lang, strings.TrimSuffix(imp, "._"), from, resolved)
	}

	// if this has a parent, try parent
	lastDot := strings.LastIndex(imp, ".")
	if lastDot > 0 {
		parent := imp[0:lastDot]
		return resolveImport(c, ix, registry, origin, lang, parent, from, resolved)
	}

	// we are down to a single symbol now.  Probe the importRegistry for a
	// type in our package.
	if origin.SourceFile != nil {
		got, provider := findPackageSymbolCompletion(registry, origin.SourceFile.Packages, imp)
		if got != "" {
			origin.Actual = imp
			resolved.Set(provider, imp, origin)
			return []label.Label{provider}
		}
	}

	return nil
}

// resolveAnyKind answers the question "what bazel label provides a rule for the
// given import?" (having the same rule kind as the given rule argument).  The
// algorithm first consults the override list (configured either via gazelle
// resolve directives, or via a YAML config).  If no override is found, the
// RuleIndex is consulted, which contains all rules indexed by gazelle in the
// generation phase.
func resolveAnyKind(c *config.Config, ix *resolve.RuleIndex, lang string, imp string, from label.Label) []label.Label {
	if l, ok := resolve.FindRuleWithOverride(c, resolve.ImportSpec{Lang: lang, Imp: imp}, ScalaLangName); ok {
		// log.Println(from, "override hit:", l)
		return []label.Label{l}
	}
	return resolveWithIndex(c, ix, lang, imp, from)
}

func resolveWithIndex(c *config.Config, ix *resolve.RuleIndex, kind, imp string, from label.Label) []label.Label {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: kind, Imp: imp}, ScalaLangName)
	if len(matches) == 0 {
		return nil
	}
	labels := make([]label.Label, len(matches))
	for i, match := range matches {
		if match.IsSelfImport(from) {
			labels[i] = PlatformLabel
		} else {
			labels[i] = match.Label
		}
	}
	return labels
}

// isSameImport returns true if the "from" and "to" labels are the same,
// normalizing to the config.RepoName.
func isSameImport(sc *scalaConfig, kind string, from, to label.Label) bool {
	if from.Repo == "" {
		from.Repo = sc.config.RepoName
	}
	if to.Repo == "" {
		to.Repo = sc.config.RepoName
	}
	if mapping, ok := sc.mapKindImportNames[kind]; ok {
		from = mapping.Rename(from)
	}
	return from == to
}

// StripRel removes the rel prefix from a filename (if has matching prefix)
func StripRel(rel string, filename string) string {
	if !strings.HasPrefix(filename, rel) {
		return filename
	}
	filename = filename[len(rel):]
	return strings.TrimPrefix(filename, "/")
}

func printRules(rules ...*rule.Rule) {
	file := rule.EmptyFile("", "")
	for _, r := range rules {
		r.Insert(file)
	}
	fmt.Println(string(file.Format()))
}

func getScalaImportsFromRuleComment(r *rule.Rule) (imports []string) {
	for _, line := range r.Comments() {
		fields := strings.Fields(line)
		// ["#", "scala-import:", "org.json4s.CustomSerializer"]
		if len(fields) < 3 {
			continue
		}
		if fields[1] != "scala-import:" {
			continue
		}
		imports = append(imports, fields[2])
	}
	return
}

// dedupLabels deduplicates labels but keeps existing ordering.
func dedupLabels(in []label.Label) (out []label.Label) {
	seen := make(map[label.Label]bool)
	for _, l := range in {
		if seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

// importStack is a simple stack of strings.
type importStack []string

func (s importStack) push(v ...string) importStack {
	return append(s, v...)
}

func (s importStack) empty() bool {
	return len(s) == 0
}

func (s importStack) pop() (importStack, string) {
	l := len(s)
	return s[:l-1], s[l-1]
}
