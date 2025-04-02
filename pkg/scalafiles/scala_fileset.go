package semanticdb

import (
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	ScalaFilesetRuleKind = "scala_fileset"
	ScalaFilesetRuleLoad = "@build_stack_scala_gazelle//rules:scala_files.bzl"
)

var (
	// deps holds the list of final deps for the aggregating rule.  This value
	// is modified as a package global.
	deps = make([]string, 0)
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		provider := NewScalaFilesetRuleProvider(load, kind)
		if err := scalarule.GlobalProviderRegistry().RegisterProvider(fqn, provider); err != nil {
			log.Fatalf("registering %s rule provider: %v", ScalaFilesetRuleKind, err)
		}
	}
	mustRegister(ScalaFilesetRuleLoad, ScalaFilesetRuleKind)
}

func NewScalaFilesetRuleProvider(load, kind string) *ScalaFilesetRuleProvider {
	return &ScalaFilesetRuleProvider{load, kind}
}

// ScalaFilesetRuleProvider implements a scalarule.Provider for the semanticdb_index.
type ScalaFilesetRuleProvider struct {
	load, kind string
}

// Name implements part of the scalarule.Provider interface.
func (s *ScalaFilesetRuleProvider) Name() string {
	return s.kind
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *ScalaFilesetRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		NonEmptyAttrs: map[string]bool{
			"deps": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *ScalaFilesetRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.kind},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.  It always
// returns nil.  The ResolveRule interface is the intended use case.
func (s *ScalaFilesetRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	// we only generate a new rule if this is a root BUILD file
	if pkg.GenerateArgs().Rel != "" {
		return nil
	}
	r := rule.NewRule(s.kind, s.kind)
	return &scalaFilesetRule{cfg, pkg, r}
}

// ResolveRule implements the RuleResolver interface.
func (s *ScalaFilesetRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {
	r.SetPrivateAttr(config.GazelleImportsKey, nil) // ensure the rule has the key, but we don't use a value
	return &scalaFilesetRule{cfg, pkg, r}
}

// scalaFilesetRule implements scalarule.RuleProvider for existing scala rules.
type scalaFilesetRule struct {
	cfg  *scalarule.Config
	pkg  scalarule.Package
	rule *rule.Rule
}

// Kind implements part of the ruleProvider interface.
func (s *scalaFilesetRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaFilesetRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaFilesetRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the scalarule.RuleProvider interface.  It always
// returns nil as this is not an importable rule.
func (s *scalaFilesetRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return nil
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *scalaFilesetRule) Resolve(rctx *scalarule.ResolveContext, importsRaw interface{}) {
	sort.Strings(deps)
	rctx.Rule.SetAttr("deps", deps)
}
