package semanticdb

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	semanticdbIndexRuleKind = "semanticdb_index"
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		provider := &semanticdbIndexRuleProvider{load, kind}
		if err := scalarule.GlobalProviderRegistry().RegisterProvider(fqn, provider); err != nil {
			log.Fatalf("registering %s rule provider: %v", semanticdbIndexRuleKind, err)
		}
	}
	mustRegister("@build_stack_scala_gazelle//rules:semanticdb_index.bzl", "semanticdb_index")
}

// semanticdbIndexRuleProvider implements a scalarule.Provider for the semanticdb_index.
type semanticdbIndexRuleProvider struct {
	load, name string
}

// Name implements part of the scalarule.Provider interface.
func (s *semanticdbIndexRuleProvider) Name() string {
	return s.name
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *semanticdbIndexRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *semanticdbIndexRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.  It always
// returns nil.  The ResolveRule interface is the intended use case.
func (s *semanticdbIndexRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *semanticdbIndexRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {

	r.SetPrivateAttr(config.GazelleImportsKey, nil)

	return &semanticdbIndexRule{cfg, pkg, r}
}

// semanticdbIndexRule implements scalarule.RuleProvider for existing scala rules.
type semanticdbIndexRule struct {
	cfg  *scalarule.Config
	pkg  scalarule.Package
	rule *rule.Rule
}

// Kind implements part of the ruleProvider interface.
func (s *semanticdbIndexRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *semanticdbIndexRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *semanticdbIndexRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the scalarule.RuleProvider interface.
func (s *semanticdbIndexRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return nil
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *semanticdbIndexRule) Resolve(rctx *scalarule.ResolveContext, importsRaw interface{}) {
	log.Println("Resolve!")
}
