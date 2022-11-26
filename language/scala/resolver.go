package scala

import (
	"log"
	"strings"
	"unicode"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// resolverImpLangPrivateKey stores the implementation language override.
const resolverImpLangPrivateKey = "_resolve_imp_lang"

// debug is a developer setting
const debug = false

// shouldDisambiguate is a developer flag
const shouldDisambiguate = false

func resolveImports(c *config.Config, ix *resolve.RuleIndex, importRegistry ScalaImportRegistry, impLang, kind string, from label.Label, imports ImportOriginMap, resolved LabelImportMap) {
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
				log.Println(from, "| resolve miss:", imp, "to", labels)
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
				if dbg {
					log.Println(from, "| resolve hit:", imp, "to", dep, "via", origin)
				}
				resolved.Set(dep, imp, origin)
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

	// if this is a fqcn, try the package
	lastDot := strings.LastIndex(imp, ".")
	if lastDot > 0 {
		child := imp[lastDot+1:]
		log.Println(from, "| resolveImport parent:", imp, "child:", child)

		if isCapitalized(child) {
			parent := imp[0:lastDot]
			return resolveImport(c, ix, registry, origin, lang, parent, from, resolved)
		}
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
	log.Println(from, "| resolveWithIndex: found rules by import with config:", imp, "->", labels)
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

func isCapitalized(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
		break
	}
	return true
}
