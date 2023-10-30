package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		if err := scalarule.
			GlobalProviderRegistry().
			RegisterProvider(fqn, &existingScalaRuleProvider{load, kind}); err != nil {
			log.Fatalf("registering scala_rule providers: %v", err)
		}
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test")
}

// existingScalaRuleProvider implements RuleResolver for scala-like rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is used to parse files named in 'srcs' and update 'deps'.
type existingScalaRuleProvider struct {
	load, name string
}

// Name implements part of the scalarule.Provider interface.
func (s *existingScalaRuleProvider) Name() string {
	return s.name
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *existingScalaRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *existingScalaRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.  It always
// returns nil.  The ResolveRule interface is the intended use case.
func (s *existingScalaRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *existingScalaRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {
	scalaRule, err := pkg.ParseRule(r, "srcs")
	if err != nil {
		if err == scalarule.ErrRuleHasNoSrcs {
			return nil // no need to print a warning
		}
		log.Printf("skipping %s %s: unable to collect srcs: %v", r.Kind(), r.Name(), err)
		return nil
	}
	if scalaRule == nil {
		log.Panicln("scalaRule should not be nil!")
	}

	r.SetPrivateAttr(config.GazelleImportsKey, scalaRule)

	return &existingScalaRule{cfg, pkg, r, scalaRule}
}

// existingScalaRule implements scalarule.RuleProvider for existing scala rules.
type existingScalaRule struct {
	cfg       *scalarule.Config
	pkg       scalarule.Package
	rule      *rule.Rule
	scalaRule scalarule.Rule
}

// Kind implements part of the ruleProvider interface.
func (s *existingScalaRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *existingScalaRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *existingScalaRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the scalarule.RuleProvider interface.
func (s *existingScalaRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return s.scalaRule.Exports()
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *existingScalaRule) Resolve(rctx *scalarule.ResolveContext, importsRaw interface{}) {
	scalaRule, ok := importsRaw.(*scalaRule)
	if !ok {
		return
	}

	imports := scalaRule.Resolve(rctx)
	r := rctx.Rule
	sc := getScalaConfig(rctx.Config)

	// part 1: deps

	deps := sc.cleanDeps(r.Attr("deps"))
	sc.mergeDeps(r.Kind(), deps, imports.Deps(sc.maybeRewrite(r.Kind(), rctx.From)))
	r.SetAttr("deps", deps)

	// part 2: srcs

	if sc.shouldAnnotateImports() {
		comments := r.AttrComments("srcs")
		if comments != nil {
			annotateImports(imports, comments)
		}
	}
}

func annotateImports(imports resolver.ImportMap, comments *build.Comments) {
	comments.Before = nil
	for _, key := range imports.Keys() {
		imp := imports[key]
		comments.Before = append(comments.Before, imp.Comment())
	}
}
