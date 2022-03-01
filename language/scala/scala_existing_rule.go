package scala

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
type scalaExistingRule struct{ load, name string }

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
		// SubstituteAttrs: map[string]bool{
		// 	"deps": true,
		// },
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
	srcs, err := getAttrFiles(pkg, existing, "srcs")
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", existing.Kind(), pkg.Rel(), existing.Name(), err)
		return nil
	}

	// If we cannot find any srcs for the rule, skip it.
	if len(srcs) == 0 {
		log.Printf("skipping %s //%s:%s (no srcs)", existing.Kind(), pkg.Rel(), existing.Name())
		return nil
	}

	from := label.New("", pkg.Rel(), existing.Name())

	requires, provides, err := resolveSrcsSymbols(pkg.Dir(), from, existing.Kind(), srcs, pkg.ScalaFileParser())
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", existing.Kind(), pkg.Rel(), existing.Name(), err)
		return nil
	}

	if debug {
		log.Println(from, "requires:", requires)

		for i, src := range srcs {
			log.Println(from, "srcs:", i, src)
		}
		for i, v := range requires {
			log.Println(from, "requires:", i, v)
		}
		for i, v := range provides {
			log.Println(from, "provides:", i, v)
		}
	}

	existing.SetPrivateAttr(config.GazelleImportsKey, requires)
	existing.SetPrivateAttr(ResolverImpLangPrivateKey, "scala")

	if debug {
		for _, imp := range requires {
			existing.AddComment("# import: " + imp)
		}
	}

	return &scalaExistingRuleRule{cfg, pkg, existing, requires, provides}
}

// scalaExistingRuleRule implements RuleProvider for existing scala rules.
type scalaExistingRuleRule struct {
	cfg  *RuleConfig
	pkg  ScalaPackage
	rule *rule.Rule
	// requires is the list of scala symbols this group of files requires
	// (import statements).
	requires []string
	// provides is the list of scala symbols this group of files provides
	// (classes, traits, etc).
	provides []string
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
	specs := make([]resolve.ImportSpec, len(s.provides))
	for i, imp := range s.provides {
		specs[i] = resolve.ImportSpec{
			Lang: "scala",
			Imp:  imp,
		}
	}
	return specs
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaExistingRuleRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
	resolveDeps("deps", s.pkg.ScalaImportRegistry())(c, ix, r, imports, from)
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

func resolveSrcsSymbols(dir string, from label.Label, kind string, srcs []string, parser ScalaFileParser) (requires, provides []string, err error) {
	var spec index.ScalaRuleSpec
	spec, err = parser.ParseScalaFiles(dir, from, kind, srcs...)
	if err != nil {
		return
	}

	for _, file := range spec.Srcs {
		for _, imp := range file.Imports {
			// exclude imports that appear to be in-package.
			// if isUnqualifiedImport(imp) {
			// 	continue
			// }
			requires = append(requires, imp)
		}
		provides = append(provides, file.Packages...)
		provides = append(provides, file.Classes...)
		provides = append(provides, file.Objects...)
		provides = append(provides, file.Traits...)
		provides = append(provides, file.Types...)
		provides = append(provides, file.Vals...)
	}

	requires = protoc.DeduplicateAndSort(requires)
	provides = protoc.DeduplicateAndSort(provides)
	return
}

// isUnqualifiedImport examples: 'CastDepthUtils._' or 'CastDepthUtils'.
func isUnqualifiedImport(imp string) bool {
	imp = strings.TrimSuffix(imp, "._")
	return strings.LastIndex(imp, ".") == -1
}
