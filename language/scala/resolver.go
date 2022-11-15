package scala

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/emicklei/dot"
)

// resolverImpLangPrivateKey stores the implementation language override.
const resolverImpLangPrivateKey = "_resolve_imp_lang"

// debug is a developer setting
const debug = true

// shouldDisambiguate is a developer flag
const shouldDisambiguate = false

func gatherIndirectDependencies(c *config.Config, imports ImportOriginMap, g *dot.Graph) {
	sc := getScalaConfig(c)

	stack := make(importStack, 0, len(imports))
	for imp := range imports {
		stack = stack.push(imp)
	}
	var imp string
	for !stack.empty() {
		stack, imp = stack.pop()
		for _, dep := range sc.GetIndirectDependencies(ScalaLangName, imp) {
			// make this a feature tooggle? (gather transitive indirects)
			stack = stack.push(dep)
			if _, ok := imports[dep]; !ok {
				imports[dep] = &ImportOrigin{Kind: ImportKindIndirect, Parent: imp}
				src := g.Node("imp/" + imp)
				dst := g.Node("imp/" + dep)
				g.Edge(src, dst, "indirect")
			}
		}
	}
}

func gatherImplicitDependencies(c *config.Config, imports ImportOriginMap, g *dot.Graph) {
	sc := getScalaConfig(c)

	stack := make(importStack, 0, len(imports))
	for imp := range imports {
		stack = stack.push(imp)
	}
	var imp string
	for !stack.empty() {
		stack, imp = stack.pop()
		for _, dep := range sc.GetImplicitDependencies(ScalaLangName, imp) {
			stack = stack.push(dep)
			if _, ok := imports[dep]; !ok {
				imports[dep] = &ImportOrigin{Kind: ImportKindIndirect, Parent: imp}
				src := g.Node("imp/" + imp)
				dst := g.Node("imp/" + dep)
				g.Edge(src, dst, "implicit")
			}
		}
	}
}

func resolveTransitive(c *config.Config, ix *resolve.RuleIndex, importRegistry ScalaImportRegistry, impLang, kind string, from label.Label, imports ImportOriginMap, resolved LabelImportMap, g *dot.Graph) {
	// at this point we expect 'resolved' to hold a set of 'actual' imports that
	// represent concrete types. build a second set of imports to be resolved
	// from that set.
	transitiveImports := make(ImportOriginMap)

	for lbl, imps := range resolved {
		if lbl == label.NoLabel {
			continue
		}
		for _, origin := range imps {
			if origin.Actual == "" {
				log.Panicln("origin.Actual must not be empty", lbl, origin.String())
			}
			src := g.Node("imp/" + origin.Actual)
			transitive, unresolved := importRegistry.TransitiveImports([]string{"imp/" + origin.Actual}, -1)
			if debug {
				if len(unresolved) > 0 {
					log.Println(from, "| warning: unresolved transitive import:", unresolved, origin.Actual)
				}
			}
			for _, tImp := range transitive {
				if _, ok := imports[tImp]; !ok {
					transitiveImports[tImp] = &ImportOrigin{Kind: ImportKindTransitive, Parent: origin.Actual}
					dst := g.Node("imp/" + tImp)
					g.Edge(src, dst, "transitive")
				}
			}
			// log.Println(from, "transitive imports:", origin.Actual, transitive)
			origin.Children = transitive
		}
	}

	// another round of indirects
	gatherIndirectDependencies(c, transitiveImports, g)

	// finally, resolve the transitive set
	resolveImports(c, ix, importRegistry, impLang, kind, from, transitiveImports, resolved, g)
}

func resolveImports(c *config.Config, ix *resolve.RuleIndex, importRegistry ScalaImportRegistry, impLang, kind string, from label.Label, imports ImportOriginMap, resolved LabelImportMap, g *dot.Graph) {
	sc := getScalaConfig(c)

	dbg := false
	for imp, origin := range imports {
		src := g.Node("imp/" + imp)

		if dbg {
			log.Println("---", from, imp, "---")
			// log.Println("resolved:\n", resolved.String())
		}

		labels := resolveImport(c, ix, importRegistry, origin, impLang, imp, from, resolved)

		if imp != origin.Actual {
			dst := g.Node("imp/" + origin.Actual)
			g.Edge(src, dst, "actual")
			src = dst
		}

		if len(labels) == 0 {
			resolved[label.NoLabel][imp] = origin
			if dbg {
				log.Println("resolved:", imp)
			}
			continue
		}

		if shouldDisambiguate && len(labels) > 1 {
			original := labels
			disambiguated, err := importRegistry.Disambiguate(c, ix, resolve.ImportSpec{Lang: ScalaLangName, Imp: imp}, ScalaLangName, from, labels)
			if err != nil {
				log.Printf("disambigation error: %v", err)
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
				dst := g.Node("rule/" + dep.String())
				g.Edge(src, dst, "label")
			}
		}
	}
}

func resolveImport(c *config.Config, ix *resolve.RuleIndex, registry ScalaImportRegistry, origin *ImportOrigin, lang string, imp string, from label.Label, resolved LabelImportMap) []label.Label {
	// if the import is empty, we may have reached the root symbol.
	if imp == "" {
		return nil
	}

	if debug {
		log.Println(from, "| resolveImport want:", imp, origin.String())
	}

	labels := resolveAnyKind(c, ix, lang, imp, from)

	if len(labels) > 0 {
		origin.Actual = imp
		if debug {
			log.Println(from, "| resolveImport got:", imp, "(provided-by)", labels)
		}
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
		log.Println(from, "| resolveAnyKind: found rule with override:", l)
		return []label.Label{l}
	}
	return resolveWithIndex(c, ix, lang, imp, from)
}

func resolveWithIndex(c *config.Config, ix *resolve.RuleIndex, kind, imp string, from label.Label) []label.Label {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: kind, Imp: imp}, ScalaLangName)
	if len(matches) == 0 {
		log.Println(from, "| resolveWithIndex: no rules found for:", imp)
		return nil
	}
	labels := make([]label.Label, 0, len(matches))
	for _, match := range matches {
		if match.IsSelfImport(from) {
			labels = append(labels, PlatformLabel)
		} else {
			labels = append(labels, match.Label)
		}
		for _, directDep := range match.Embeds {
			labels = append(labels, directDep)
		}
	}
	log.Println(from, "| resolveWithIndex: found rules by import with config:", labels)
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

func getScalaImportsFromRuleAttrComment(attrName, prefix string, r *rule.Rule) (imports []string) {
	// assign := r.AttrAssignment(attrName)
	var assign *build.AssignExpr
	if assign == nil {
		return
	}

	for _, comment := range assign.Before {
		line := comment.Token
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, prefix))
		dblslash := strings.Index(line, "//")
		if dblslash != -1 {
			line = strings.TrimSpace(line[:dblslash])
		}
		imports = append(imports, line)
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
