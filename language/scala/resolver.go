package scala

import (
	"fmt"
	"log"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/index"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	// ResolverImpLangPrivateKey stores the implementation language override.
	ResolverImpLangPrivateKey = "_resolve_imp_lang"
	debug                     = true
)

type labelImportMap map[label.Label]map[string]bool

func (m labelImportMap) Set(from label.Label, imp string) {
	if all, ok := m[from]; ok {
		all[imp] = true
	} else {
		m[from] = map[string]bool{imp: true}
	}
}

func resolveImport(c *config.Config, ix *resolve.RuleIndex, registry ScalaImportRegistry, file *index.ScalaFileSpec, lang string, imp string, from label.Label, labelMap labelImportMap) []label.Label {
	if debug {
		log.Println("resolveImport:", imp)
	}

	// if the import is empty, we may have reached the root symbol.
	if imp == "" {
		return nil
	}

	labels := resolveAnyKind(c, ix, lang, imp, from)
	if debug {
		log.Println("resolveAnyKind:", imp, labels)
	}

	if len(labels) > 1 {
		labels = dedupLabels(labels)
	}
	if len(labels) > 0 {
		for _, l := range labels {
			labelMap.Set(l, imp)
		}
		return labels
	}

	// if this is a _root_ import, try without
	if strings.HasPrefix(imp, "_root_.") {
		return resolveImport(c, ix, registry, file, lang, strings.TrimPrefix(imp, "_root_."), from, labelMap)
	}

	// if this is a wildcard import, try without
	if strings.HasSuffix(imp, "._") {
		return resolveImport(c, ix, registry, file, lang, strings.TrimSuffix(imp, "._"), from, labelMap)
	}

	// if this has a parent, try parent
	lastDot := strings.LastIndex(imp, ".")
	if lastDot > 0 {
		parent := imp[0:lastDot]
		return resolveImport(c, ix, registry, file, lang, parent, from, labelMap)
	}

	// we are down to a single symbol now.  Probe the importRegistry for a
	// type in our package.
	if file != nil {
		for _, pkg := range file.Packages {
			log.Printf("probing for %q in package %s", imp, pkg)
			completions := registry.Completions(pkg)
			for actualType, provider := range completions {
				if imp == actualType {
					log.Printf("matched in-package import=%s.%s: %v", pkg, imp, provider)
					labelMap.Set(provider, imp)
					return []label.Label{provider}
				}
			}
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
		log.Println(from, "override hit:", l)
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
			labels[i] = label.NoLabel
		} else {
			labels[i] = match.Label
		}
	}
	return labels
}

// isSameImport returns true if the "from" and "to" labels are the same.  If the
// "to" label is not a canonical label (having a fully-qualified repo name), a
// canonical label is constructed for comparison using the config.RepoName.
func isSameImport(c *config.Config, from, to label.Label) bool {
	if from == to {
		return true
	}
	if to.Repo != "" {
		return false
	}
	canonical := label.New(c.RepoName, to.Pkg, to.Name)
	return from == canonical
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
		if len(fields) != 3 {
			continue
		}
		if fields[1] != "scala-import:" {
			continue
		}
		imports = append(imports, fields[2])
		// log.Println("add scala import from rule comment:", fields[2])
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
		if l == label.NoLabel || l == PlatformLabel {
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
