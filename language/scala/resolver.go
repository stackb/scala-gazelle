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
	"github.com/bazelbuild/buildtools/build"
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

type depsResolver func(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label)

func resolveDeps(attrName string) depsResolver {
	return func(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
		if debug {
			log.Printf("resolveDeps %q for %s rule %v", attrName, r.Kind(), from)
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

		for _, imp := range imports {

			if debug {
				log.Println("resolveDeps:", impLang, imp)
			}

			l, err := resolveImport(c, ix, impLang, imp, from)
			if err == errSkipImport {
				if debug {
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
			if l == label.NoLabel {
				unresolved = append(unresolved, "no-label: "+imp)
				if debug {
					log.Println(from, "no label", imp)
				}
				continue
			}

			l = l.Rel(from.Repo, from.Pkg)
			if debug {
				log.Println(from, "resolved:", imp, "is provided by", l)
			}
			depSet[l.String()] = true
			resolved = append(resolved, imp+" -> "+l.String())
		}

		if len(depSet) > 0 {
			deps := make([]string, 0, len(depSet))
			for dep := range depSet {
				deps = append(deps, dep)
			}
			sort.Strings(deps)
			r.SetAttr(attrName, deps)
			if debug {
				log.Println(from, "resolved deps:", deps)
				printRules(r)
			}

			if len(unresolved) > 0 {
				if true {
					panic(fmt.Sprintf("unresolved deps! %v", unresolved))
				}
				unresolved = protoc.DeduplicateAndSort(unresolved)
				before := make([]build.Comment, len(unresolved))
				for i, imp := range unresolved {
					before[i].Token = "# unresolved" + imp
				}
				r.Attr(attrName).Comment().After = before
			} else {
				resolved = protoc.DeduplicateAndSort(resolved)
				before := make([]build.Comment, len(resolved))
				for i, imp := range resolved {
					before[i].Token = "# resolved: " + imp
				}

				r.Attr(attrName).Comment().After = before
				// log.Println(from, "resolved deps:", resolved)
			}

		}
	}
}

func resolveImport(c *config.Config, ix *resolve.RuleIndex, lang string, imp string, from label.Label) (label.Label, error) {
	if debug {
		log.Println("resolveImport", from, lang, imp)
	}
	// if the import is empty, we may have reached the root symbol.
	if imp == "" {
		return label.NoLabel, errSkipImport
	}
	l, err := resolveAnyKind(c, ix, lang, imp, from)
	if l == PlatformLabel {
		return l, errSkipImport
	}
	if l == label.NoLabel {
		// if this is already a package import, try the parent package
		imp = strings.TrimSuffix(imp, "._")
		lastDot := strings.LastIndex(imp, ".")
		if lastDot > 0 {
			parentPkg := imp[0:lastDot] + "._"
			if debug {
				log.Println("resolveImport parent package", from, lang, parentPkg)
			}
			return resolveImport(c, ix, lang, parentPkg, from)
		}
	}
	return l, err
}

// resolveAnyKind answers the question "what bazel label provides a rule for the
// given import?" (having the same rule kind as the given rule argument).  The
// algorithm first consults the override list (configured either via gazelle
// resolve directives, or via a YAML config).  If no override is found, the
// RuleIndex is consulted, which contains all rules indexed by gazelle in the
// generation phase.   If no match is found, return label.NoLabel.
func resolveAnyKind(c *config.Config, ix *resolve.RuleIndex, lang string, imp string, from label.Label) (label.Label, error) {
	if l, ok := resolve.FindRuleWithOverride(c, resolve.ImportSpec{Lang: lang, Imp: imp}, ScalaLangName); ok {
		// log.Println(from, "override hit:", l)
		return l, nil
	}
	if l, err := resolveWithIndex(c, ix, lang, imp, from); err == nil || err == errSkipImport {
		return l, err
	} else if err != errNotFound {
		return label.NoLabel, err
	}
	return label.NoLabel, nil
}

func resolveWithIndex(c *config.Config, ix *resolve.RuleIndex, kind, imp string, from label.Label) (label.Label, error) {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: kind, Imp: imp}, ScalaLangName)
	if len(matches) == 0 {
		// log.Println(from, "no matches:", imp)
		return label.NoLabel, errNotFound
	}
	if len(matches) > 1 {
		return label.NoLabel, fmt.Errorf("multiple rules (%s and %s) may be imported with %q from %s", matches[0].Label, matches[1].Label, imp, from)
	}
	if matches[0].IsSelfImport(from) || isSameImport(c, from, matches[0].Label) {
		// log.Println(from, "self import:", imp)
		return label.NoLabel, errSkipImport
	}
	// log.Println(from, "FindRulesByImportWithConfig first match:", imp, matches[0].Label)
	return matches[0].Label, nil
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
