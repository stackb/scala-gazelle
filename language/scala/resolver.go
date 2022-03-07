package scala

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/rules_proto/pkg/protoc"
)

const (
	// ResolverImpLangPrivateKey stores the implementation language override.
	ResolverImpLangPrivateKey = "_resolve_imp_lang"
)

var (
	debug         = false
	errSkipImport = errors.New("self import")
	errNotFound   = errors.New("rule not found")
)

type depsResolver func(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports interface{}, from label.Label)

func resolveDeps(attrName string, importRegistry ScalaImportRegistry) depsResolver {
	return func(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
		imports, ok := importsRaw.([]string)
		if !ok {
			return
		}
		dbg := debug
		if dbg {
			log.Printf("resolveDeps %q for %s rule %v", attrName, r.Kind(), from)
		}

		sc := getScalaConfig(c)

		if from.Name == "" {
			log.Panicf("resolver: bad from label: %v", from)
		}

		existing := r.AttrStrings(attrName)
		r.DelAttr(attrName)

		depSet := make(map[string]bool)
		for _, d := range existing {
			depSet[d] = true
		}

		unresolved := make([]string, 0)
		resolved := make([]string, 0)

		// determine the resolve kind
		impLang := r.Kind()
		if overrideImpLang, ok := r.PrivateAttr(ResolverImpLangPrivateKey).(string); ok {
			impLang = overrideImpLang
		}

		if dbg {
			log.Printf("resolving %d imports: %v", len(imports), imports)
		}

		stack := make(importStack, 0)
		stack = stack.push(imports...)
		stack = stack.push(getScalaImportsFromRuleComment(r)...)

		var imp string

		for !stack.empty() {
			stack, imp = stack.pop()

			// push any indirect dependencies
			if deps := sc.GetIndirectDependencies(ScalaLangName, imp); len(deps) > 0 {
				if dbg {
					log.Println("adding indirect deps", deps)
				}
				stack = stack.push(deps...)
			}

			if dbg {
				log.Println("---", imp, "---")
			}
			ll, err := resolveImport(c, ix, impLang, imp, from)
			if err == errSkipImport {
				if dbg {
					log.Println(from, "skipped:", imp)
				}
				// Note: skipped imports do not contribute to 'unresolved' list.
				continue
			}
			if err != nil {
				log.Println(from, "scala resolveDeps error:", err)
				unresolved = append(unresolved, "error: "+imp+": "+err.Error())
				continue
			}
			if len(ll) == 0 {
				unresolved = append(unresolved, "no-label: "+imp)
				log.Panicln(from, "unresolved import (no label):", imp)
				continue
			}
			if len(ll) > 1 {
				original := ll
				disambiguated, err := importRegistry.Disambiguate(c, imp, ll, from)
				if err != nil {
					log.Fatalf("error while disambiguating %q %v (from=%v): %v", imp, ll, from, err)
				}
				if false {
					if len(ll) > 0 {
						if strings.HasSuffix(imp, "._") {
							log.Fatalf("%v: %q is ambiguous. Use a 'gazelle:resolve' directive, refactor the class without a wildcard import, or manually add deps with '# keep' comments): %v", from, imp, ll)
						} else {
							log.Fatalf("%v: %q is ambiguous. Use a 'gazelle:resolve' directive, refactor the class, or manually add deps with '# keep' comments): %v", from, imp, ll)
						}
					}
				}
				ll = disambiguated
				resolved = append(resolved, fmt.Sprintf("diambiguated %q: %v => %v", imp, original, disambiguated))
			}
			for _, l := range ll {
				// one final check for self imports
				if from.Equal(l) || isSameImport(c, from, l) {
					continue
				}
				l = l.Rel(from.Repo, from.Pkg)
				if dbg {
					log.Println(from, "resolved:", imp, "is provided by", l)
				}
				depSet[l.String()] = true
				resolved = append(resolved, imp+" -> "+l.String())
			}
		}

		if len(depSet) > 0 {
			deps := make([]string, 0, len(depSet))
			for dep := range depSet {
				deps = append(deps, dep)
			}
			sort.Strings(deps)
			r.SetAttr(attrName, deps)

			if true {
				tags := r.AttrStrings("tags")
				tags = append(tags, protoc.DeduplicateAndSort(resolved)...)
				r.SetAttr("tags", tags)
			}

			if len(unresolved) > 0 {
				if true {
					panic(fmt.Sprintf("unresolved deps! %v", unresolved))
				}
				r.SetAttr("unresolved_deps", protoc.DeduplicateAndSort(unresolved))
			}

		}
	}
}

func resolveImport(c *config.Config, ix *resolve.RuleIndex, lang string, imp string, from label.Label) ([]label.Label, error) {
	if debug {
		log.Println("resolveImport", from, lang, imp)
	}
	// if the import is empty, we may have reached the root symbol.
	if imp == "" {
		return nil, errSkipImport
	}

	ll, err := resolveAnyKind(c, ix, lang, imp, from)
	if err != nil {
		return nil, err
	}

	if len(ll) == 1 && ll[0] == PlatformLabel {
		return nil, errSkipImport
	}

	if len(ll) == 0 {
		// if this is a _root_ import, try without
		if strings.HasPrefix(imp, "_root_.") {
			return resolveImport(c, ix, lang, strings.TrimPrefix(imp, "_root_."), from)
		}

		// if this is already an all import, try without
		if strings.HasSuffix(imp, "._") {
			return resolveImport(c, ix, lang, strings.TrimSuffix(imp, "._"), from)
		}

		lastDot := strings.LastIndex(imp, ".")
		if lastDot > 0 {
			parent := imp[0:lastDot]
			if debug {
				log.Println("resolveImport parent package", from, lang, parent)
			}
			return resolveImport(c, ix, lang, parent, from)
		}
	}

	if len(ll) > 1 {
		ll = dedupLabels(ll)
	}

	return ll, err
}

func getScalaImportsFromRuleComment(r *rule.Rule) (imports []string) {
	for _, line := range r.Comments() {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[0] != "scala-import:" {
			continue
		}
		imports = append(imports, fields[1])
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

// resolveAnyKind answers the question "what bazel label provides a rule for the
// given import?" (having the same rule kind as the given rule argument).  The
// algorithm first consults the override list (configured either via gazelle
// resolve directives, or via a YAML config).  If no override is found, the
// RuleIndex is consulted, which contains all rules indexed by gazelle in the
// generation phase.   If no match is found, return label.NoLabel.
func resolveAnyKind(c *config.Config, ix *resolve.RuleIndex, lang string, imp string, from label.Label) ([]label.Label, error) {
	if l, ok := resolve.FindRuleWithOverride(c, resolve.ImportSpec{Lang: lang, Imp: imp}, ScalaLangName); ok {
		// log.Println(from, "override hit:", l)
		return []label.Label{l}, nil
	}
	if ll, err := resolveWithIndex(c, ix, lang, imp, from); err == nil || err == errSkipImport {
		return ll, err
	} else if err != errNotFound {
		return nil, err
	}
	return nil, nil
}

func resolveWithIndex(c *config.Config, ix *resolve.RuleIndex, kind, imp string, from label.Label) ([]label.Label, error) {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: kind, Imp: imp}, ScalaLangName)
	if len(matches) == 0 {
		// log.Println(from, "no matches:", imp)
		return nil, errNotFound
	}
	if len(matches) > 1 {
		// return label.NoLabel, fmt.Errorf("%v: %q is provided by multiple rules (%s and %s).  Add a resolve directive in the nearest BUILD.bazel file to disambiguate (example: '# gazelle:resolve scala scala %s %s')", from, imp, matches[0].Label, matches[1].Label, imp, matches[0].Label)
		ll := make([]label.Label, len(matches))
		for i, match := range matches {
			// TODO(pcj): check for self imports
			ll[i] = match.Label
		}
		return ll, nil
	}
	if matches[0].IsSelfImport(from) || isSameImport(c, from, matches[0].Label) {
		// log.Println(from, "self import:", imp)
		return nil, errSkipImport
	}
	// log.Println(from, "FindRulesByImportWithConfig first match:", imp, matches[0].Label)
	return []label.Label{matches[0].Label}, nil
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
