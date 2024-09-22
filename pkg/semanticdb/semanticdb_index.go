package semanticdb

import (
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	SemanticdbIndexRuleKind = "semanticdb_index"
	SemanticdbIndexRuleLoad = "@build_stack_scala_gazelle//rules:semanticdb_index.bzl"
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		provider := NewSemanticdbIndexRuleProvider(load, kind)
		if err := scalarule.GlobalProviderRegistry().RegisterProvider(fqn, provider); err != nil {
			log.Fatalf("registering %s rule provider: %v", SemanticdbIndexRuleKind, err)
		}
	}
	mustRegister(SemanticdbIndexRuleLoad, SemanticdbIndexRuleKind)
}

func NewSemanticdbIndexRuleProvider(load, kind string) *SemanticdbIndexRuleProvider {
	return &SemanticdbIndexRuleProvider{load, kind}
}

// SemanticdbIndexRuleProvider implements a scalarule.Provider for the semanticdb_index.
type SemanticdbIndexRuleProvider struct {
	load, name string
}

// Name implements part of the scalarule.Provider interface.
func (s *SemanticdbIndexRuleProvider) Name() string {
	return s.name
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *SemanticdbIndexRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{
			"deps": true,
			"jars": true,
		},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *SemanticdbIndexRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.  It always
// returns nil.  The ResolveRule interface is the intended use case.
func (s *SemanticdbIndexRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *SemanticdbIndexRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {

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

// Imports implements part of the scalarule.RuleProvider interface.  It always
// returns nil as semanticdb_index is not an importable rule.
func (s *semanticdbIndexRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return nil
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *semanticdbIndexRule) Resolve(rctx *scalarule.ResolveContext, importsRaw interface{}) {

	kinds := make(map[string]bool)
	for _, kind := range rctx.Rule.AttrStrings("kinds") {
		kinds[kind] = true
	}

	symbols := make(map[label.Label]*resolver.Symbol)
	for _, sym := range GetGlobalScope().GetSymbols("") {
		symbols[sym.Label] = sym
	}

	deps := make([]string, 0, len(symbols))
	for lbl, sym := range symbols {
		if lbl == label.NoLabel {
			continue
		}
		if lbl.Repo != "" {
			continue
		}
		if _, ok := kinds[sym.Provider]; !ok {
			continue
		}
		dep := label.New(lbl.Repo, lbl.Pkg, lbl.Name+"_semanticdb")
		deps = append(deps, dep.String())
	}
	sort.Strings(deps)

	rctx.Rule.SetAttr("deps", deps)
}
