package scala

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/rules_proto/pkg/protoc"
	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		Rules().MustRegisterRule(fqn, &scalaExistingRule{load, kind})
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test")

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "_scala_library")
	mustRegister("//bazel_tools:scala.bzl", "scala_app")
	mustRegister("//bazel_tools:scala.bzl", "scala_app_test")
	mustRegister("//bazel_tools:scala.bzl", "scala_app_library")
	mustRegister("//bazel_tools:scala.bzl", "trumid_scala_library")
	mustRegister("//bazel_tools:scala.bzl", "trumid_scala_test")
	mustRegister("//bazel_tools:scala.bzl", "classic_scala_app")
	mustRegister("//bazel_tools:scala.bzl", "scala_e2e_app")
	mustRegister("//bazel_tools:scala.bzl", "scala_e2e_test")
}

// scalaExistingRule implements RuleResolver for scala-kind rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is to parse files named in 'srcs' and update 'deps'.
type scalaExistingRule struct {
	load, name string
}

// Name implements part of the RuleInfo interface.
func (s *scalaExistingRule) Name() string {
	return s.name
}

// KindInfo implements part of the RuleInfo interface.
func (s *scalaExistingRule) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		// TODO(pcj): understand better why deps needs to be in MergeableAttrs
		// here rather than ResolveAttrs.
		MergeableAttrs: map[string]bool{
			"deps": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	}
}

// LoadInfo implements part of the RuleInfo interface.
func (s *scalaExistingRule) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the RuleInfo interface.  It always returns
// nil.  The ResolveRule interface is the intended use case.
func (s *scalaExistingRule) ProvideRule(cfg *RuleConfig, pkg ScalaPackage) RuleProvider {
	return nil
}

// ResolveRule implement the RuleResolver interface.  It will attempt to parse
// imports and resolve deps.
func (s *scalaExistingRule) ResolveRule(cfg *RuleConfig, pkg ScalaPackage, existing *rule.Rule) RuleProvider {
	// TODO(pcj): is this rule copying actually necessary?  Introduced this when
	// trying to debug a bug where I wasn't sure if state inside the rule was
	// the issue.
	r := rule.NewRule(existing.Kind(), existing.Name())
	for _, key := range existing.AttrKeys() {
		r.SetAttr(key, existing.Attr(key))
	}
	r.DelAttr("deps") // make sure the "source" rule has no deps to start

	srcs, err := getAttrFiles(pkg, r, "srcs")
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
		return nil
	}

	// If we cannot find any srcs for the rule, skip it.
	if len(srcs) == 0 {
		log.Printf("skipping %s //%s:%s (no srcs)", r.Kind(), pkg.Rel(), r.Name())
		return nil
	}

	from := label.New("", pkg.Rel(), r.Name())

	files, err := resolveScalaSrcs(pkg.Dir(), from, r.Kind(), srcs, pkg.ScalaFileParser())
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
		return nil
	}

	r.SetPrivateAttr(config.GazelleImportsKey, files)
	r.SetPrivateAttr(ResolverImpLangPrivateKey, "scala")

	return &scalaExistingRuleRule{cfg, pkg, r, files}
}

// scalaExistingRuleRule implements RuleProvider for existing scala rules.
type scalaExistingRuleRule struct {
	cfg   *RuleConfig
	pkg   ScalaPackage
	rule  *rule.Rule
	files []*index.ScalaFileSpec
}

// Kind implements part of the ruleProvider interface.
func (s *scalaExistingRuleRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaExistingRuleRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaExistingRuleRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaExistingRuleRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	provides := make([]string, 0)
	for _, file := range s.files {
		provides = append(provides, file.Packages...)
		provides = append(provides, file.Classes...)
		provides = append(provides, file.Objects...)
		provides = append(provides, file.Traits...)
		provides = append(provides, file.Types...)
		provides = append(provides, file.Vals...)
	}
	provides = protoc.DeduplicateAndSort(provides)

	specs := make([]resolve.ImportSpec, len(provides))
	for i, imp := range provides {
		specs[i] = resolve.ImportSpec{
			Lang: "scala",
			Imp:  imp,
		}
	}

	return specs
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaExistingRuleRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
	dbg := debug
	files, ok := importsRaw.([]*index.ScalaFileSpec)
	if !ok {
		return
	}
	importRegistry := s.pkg.ScalaImportRegistry()
	sc := getScalaConfig(c)

	// gather imports in a map such that we know the file that the import arose
	// from.  IN some cases (indirect deps, those provided in rule comments) the
	// file is not known (nil).
	imports := make(map[string]*index.ScalaFileSpec)

	// 1: direct imports
	for _, file := range files {
		for _, imp := range file.Imports {
			imports[imp] = file
		}
	}

	// 2: explicity named in the rule comment.
	for _, imp := range getScalaImportsFromRuleComment(r) {
		if _, ok := imports[imp]; !ok {
			imports[imp] = nil
		}
	}

	// 3: transitive of 1+2.
	stack := make(importStack, 0, len(imports))
	for k := range imports {
		stack = stack.push(k)
	}
	var imp string
	for !stack.empty() {
		stack, imp = stack.pop()
		for _, dep := range sc.GetIndirectDependencies(ScalaLangName, imp) {
			// make this is feature tooggle? for transitive indirects?
			// stack = stack.push(dep)
			if _, ok := imports[dep]; !ok {
				imports[dep] = nil
			}
		}
	}

	// want to record which imports contributed to from
	resolved := make(labelImportMap)

	depSet := make(map[label.Label]bool)
	for _, d := range r.AttrStrings("deps") {
		l, err := label.Parse(d)
		if err != nil {
			continue
		}
		depSet[l] = true
		resolved.Set(l, "<rule attr>")
	}

	unresolved := make([]string, 0)
	// resolved := make([]string, 0)

	// determine the resolve kind
	impLang := r.Kind()
	if overrideImpLang, ok := r.PrivateAttr(ResolverImpLangPrivateKey).(string); ok {
		impLang = overrideImpLang
	}

	for imp, file := range imports {
		if dbg {
			log.Println("---", from, imp, "---")
		}

		labels := resolveImport(c, ix, importRegistry, file, impLang, imp, from, resolved)

		if len(labels) == 0 {
			unresolved = append(unresolved, "no-label: "+imp)
			if dbg {
				log.Println("unresolved:", imp)
			}
			continue
		}

		if len(labels) > 1 {
			// original := labels
			disambiguated, err := importRegistry.Disambiguate(c, imp, labels, from)
			if err != nil {
				log.Fatalf("error while disambiguating %q %v (from=%v): %v", imp, labels, from, err)
			}
			if false {
				if len(labels) > 0 {
					if strings.HasSuffix(imp, "._") {
						log.Fatalf("%v: %q is ambiguous. Use a 'gazelle:resolve' directive, refactor the class without a wildcard import, or manually add deps with '# keep' comments): %v", from, imp, labels)
					} else {
						log.Fatalf("%v: %q is ambiguous. Use a 'gazelle:resolve' directive, refactor the class, or manually add deps with '# keep' comments): %v", from, imp, labels)
					}
				}
			}
			labels = disambiguated
			// resolved = append(resolved, fmt.Sprintf("diambiguated %q: %v => %v", imp, original, disambiguated))
		}

		for _, l := range labels {
			if from.Equal(l) || isSameImport(c, from, l) || l == label.NoLabel || l == PlatformLabel {
				continue
			}
			l = l.Rel(from.Repo, from.Pkg)
			depSet[l] = true
			resolved.Set(l, imp)

			// resolved = append(resolved, imp+" -> "+l.String())
			if dbg {
				log.Println("resolved:", imp, "is provided by", l)
			}
		}
	}

	r.DelAttr("deps")

	if len(depSet) > 0 {
		deps := make([]label.Label, len(depSet))
		i := 0
		for dep := range depSet {
			deps[i] = dep
			i++
			// deps = append(deps, dep.String())
		}
		sort.Slice(deps, func(i, j int) bool {
			a := deps[i]
			b := deps[j]
			return a.String() < b.String()
		})
		list := make([]build.Expr, len(deps))
		for i, dep := range deps {
			str := &build.StringExpr{Value: dep.String()}
			list[i] = str

			if imps, ok := resolved[dep]; ok {
				reasons := make([]string, 0, len(imps))
				for imp := range imps {
					reasons = append(reasons, imp)
				}
				sort.Strings(reasons)
				for _, reason := range reasons {
					str.Comments.Before = append(str.Comments.Before, build.Comment{Token: "# " + reason})
				}
			}
		}
		// r.SetAttr("deps", deps)
		r.SetAttr("deps", &build.ListExpr{List: list})

		// if true {
		// 	tags := r.AttrStrings("tags")
		// 	tags = append(tags, protoc.DeduplicateAndSort(resolved)...)
		// 	r.SetAttr("tags", tags)
		// }

		if len(unresolved) > 0 {
			if true {
				panic(fmt.Sprintf("%v has unresolved dependencies: %v", from, unresolved))
			}
			r.SetAttr("unresolved_deps", protoc.DeduplicateAndSort(unresolved))
		}
	}

	if dbg {
		log.Println("-- | ", from, "finished deps resolution.")
	}
}

// getAttrFiles returns a list of source files for the 'srcs' attribute.  Each
// value is a repo-relative path.
func getAttrFiles(pkg ScalaPackage, r *rule.Rule, attrName string) (srcs []string, err error) {
	switch t := r.Attr(attrName).(type) {
	case *build.ListExpr:
		// example: ["foo.scala", "bar.scala"]
		for _, item := range t.List {
			switch elem := item.(type) {
			case *build.StringExpr:
				srcs = append(srcs, elem.Value)
			}
		}
	case *build.CallExpr:
		// example: glob(["**/*.scala"])
		if ident, ok := t.X.(*build.Ident); ok {
			switch ident.Name {
			case "glob":
				glob := parseGlob(pkg.File(), t)
				dir := filepath.Join(pkg.Dir(), pkg.Rel())
				srcs = append(srcs, applyGlob(glob, os.DirFS(dir))...)
			default:
				err = fmt.Errorf("not attempting to resolve function call %v(): consider making this simpler", ident.Name)
			}
		} else {
			err = fmt.Errorf("not attempting to resolve call expression %+v: consider making this simpler", t)
		}
	case *build.Ident:
		// example: srcs = LIST_OF_SOURCES
		srcs, err = globalStringList(pkg.File(), t)
		if err != nil {
			err = fmt.Errorf("faile to resolve resolve identifier %q (consider inlining it): %w", t.Name, err)
		}
	case nil:
		// TODO(pcj): should this be considered an error, or normal condition?
		// err = fmt.Errorf("rule has no 'srcs' attribute")
	default:
		err = fmt.Errorf("uninterpretable 'srcs' attribute type: %T", t)
	}

	return
}

func resolveScalaSrcs(dir string, from label.Label, kind string, srcs []string, parser ScalaFileParser) ([]*index.ScalaFileSpec, error) {
	if spec, err := parser.ParseScalaFiles(dir, from, kind, srcs...); err != nil {
		return nil, err
	} else {
		return spec.Srcs, nil
	}
}

// isUnqualifiedImport examples: 'CastDepthUtils._' or 'CastDepthUtils'.
func isUnqualifiedImport(imp string) bool {
	imp = strings.TrimSuffix(imp, "._")
	return strings.LastIndex(imp, ".") == -1
}
