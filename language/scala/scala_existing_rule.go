package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

func init() {
	mustRegister := func(load, kind string, isBinaryRule bool) {
		fqn := load + "%" + kind
		if err := scalarule.
			GlobalProviderRegistry().
			RegisterProvider(fqn, &existingRuleProvider{load, kind, isBinaryRule}); err != nil {
			log.Fatalf("registering scala_rule providers: %v", err)
		}
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary", true)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test", true)
}

// existingRuleProvider implements RuleResolver for scala-like rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is used to parse files named in 'srcs' and update 'deps'.
type existingRuleProvider struct {
	load, name   string
	isBinaryRule bool
}

// Name implements part of the scalarule.Provider interface.
func (s *existingRuleProvider) Name() string {
	return s.name
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *existingRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *existingRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.  It always returns
// nil.  The ResolveRule interface is the intended use case.
func (s *existingRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *existingRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {
	scalaRule, err := pkg.ParseRule(r, "srcs")
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

// scalaExistingRuleProvider implements scalarule.RuleProvider for existing scala rules.
type scalaExistingRuleProvider struct {
	cfg          *scalarule.Config
	pkg          scalarule.Package
	rule         *rule.Rule
	scalaRule    scalarule.Rule
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

// Imports implements part of the scalarule.RuleProvider interface.
func (s *scalaExistingRuleProvider) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	// binary rules should not be deps of anything else, so we don't advertise
	// any imports. TODO(pcj): this is too simplisitic: test helpers can be used
	// by other test rules.  So we should probably use the 'impLang' to differentiate
	// generic imports ('scala') vs testonly imports ('test').
	if s.isBinaryRule {
		return nil
	}
	return s.scalaRule.Exports()
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *scalaExistingRuleProvider) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
	scalaRule, ok := importsRaw.(*scalaRule)
	if !ok {
		return
	}

	sc := getScalaConfig(c)
	imports := scalaRule.Imports()

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
