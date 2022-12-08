package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/rules_proto/pkg/protoc"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func init() {
	mustRegister := func(load, kind string, isBinaryRule bool) {
		fqn := load + "%" + kind
		Rules().MustRegisterRule(fqn, &scalaExistingRule{load, kind, isBinaryRule})
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary", true)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test", true)
}

// scalaExistingRule implements RuleResolver for scala-kind rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is used to parse files named in 'srcs' and update 'deps'.
type scalaExistingRule struct {
	load, name   string
	isBinaryRule bool
}

// Name implements part of the RuleInfo interface.
func (s *scalaExistingRule) Name() string {
	return s.name
}

// KindInfo implements part of the RuleInfo interface.
func (s *scalaExistingRule) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{"deps": true},
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

// ResolveRule implements the RuleResolver interface.
func (s *scalaExistingRule) ResolveRule(cfg *RuleConfig, pkg ScalaPackage, r *rule.Rule) RuleProvider {
	scalaRule, err := pkg.ParseScalaRule(r)
	if err != nil {
		log.Printf("skipping %s %s: unable to collect srcs: %v", r.Kind(), r.Name(), err)
		return nil
	}
	if scalaRule == nil {
		log.Panicln("scalaRule should not be nil!")
	}

	r.SetPrivateAttr(config.GazelleImportsKey, scalaRule)

	return &scalaExistingRuleProvider{cfg, pkg, r, scalaRule, s.isBinaryRule}
}

// scalaExistingRuleProvider implements RuleProvider for existing scala rules.
type scalaExistingRuleProvider struct {
	cfg          *RuleConfig
	pkg          ScalaPackage
	rule         *rule.Rule
	scalaRule    *ScalaRule
	isBinaryRule bool
}

// Kind implements part of the ruleProvider interface.
func (s *scalaExistingRuleProvider) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaExistingRuleProvider) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaExistingRuleProvider) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaExistingRuleProvider) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	// binary rules should not be deps of anything else, so we don't advertise
	// any imports. TODO(pcj): this is too simplisitic: test helpers can be used
	// by other test rules.  So we should probably use the 'impLang' to differentiate
	// generic imports ('scala') vs testonly imports ('test').
	if s.isBinaryRule {
		return nil
	}
	if len(s.scalaRule.Files) == 0 {
		return nil
	}

	provides := make([]string, 0)
	for _, file := range s.scalaRule.Files {
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
		specs[i] = resolve.ImportSpec{Lang: scalaLangName, Imp: imp}
	}

	return specs
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaExistingRuleProvider) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
	scalaRule, ok := importsRaw.(*ScalaRule)
	if !ok {
		return
	}

	sc := getScalaConfig(c)
	imports := scalaRule.Imports(sc)

	if len(imports) > 0 {
		for _, imp := range imports.Values() {
			if known, err := scalaRule.ResolveKnownImport(c, ix, from, scalaLangName, imp.Imp); err != nil {
				imp.Error = err
			} else {
				imp.Known = known
			}
		}

		deps := buildKeepDepsList(sc, r.Attr("deps"))
		addResolvedDeps(deps, sc, r.Kind(), from, imports)

		r.SetAttr("deps", deps)
	}

	if sc.shouldAnnotateImports() || sc.shouldAnnotateResolvedDeps() {
		attr := r.Attr("srcs")
		switch t := attr.(type) {
		case *build.ListExpr:
			annotateImports(imports, &t.Comments, sc.shouldAnnotateImports(), sc.shouldAnnotateUnresolvedDeps())
		case *build.CallExpr:
			annotateImports(imports, &t.Comments, sc.shouldAnnotateImports(), sc.shouldAnnotateUnresolvedDeps())
		case *build.BinaryExpr:
			annotateImports(imports, &t.Comments, sc.shouldAnnotateImports(), sc.shouldAnnotateUnresolvedDeps())
		}
	}

	// r.Attr("name").(*build.StringExpr).Comments.Before = []build.Comment{sc.Comment()}
}

func annotateImports(imports resolver.ImportMap, comments *build.Comments, wantImports, wantUnresolved bool) {
	comments.Before = nil
	for _, key := range imports.Keys() {
		imp := imports[key]
		if !(wantImports || (wantUnresolved && imp.Known == nil)) {
			continue
		}
		comments.Before = append(comments.Before, imp.Comment())
	}
}
