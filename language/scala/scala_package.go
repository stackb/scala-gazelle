package scala

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/glob"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	ruleProviderKey = "_scala_rule_provider"
)

var ErrRuleHasNoSrcs = fmt.Errorf("rule has no source files")

// scalaPackage provides a set of proto_library derived rules for the package.
type scalaPackage struct {
	// args are the generateArgs
	args language.GenerateArgs
	// parser is the file parser
	parser parser.Parser
	// universe is the parent universe
	universe resolver.Universe
	// the registry to use
	providerRegistry scalarule.ProviderRegistry
	// the config for this package
	cfg *scalaconfig.Config
	// the generated and empty rule providers
	gen, empty []scalarule.RuleProvider
	// rules is the final state of generated rules, by name.
	rules map[string]*rule.Rule
	// resolveFuncs is a list of resolve work that needs to be deferred until
	// all rules in a package have been processed.
	resolveWork []func()
	// used for tracking coverage
	ruleCoverage *packageRuleCoverage
}

// newScalaPackage constructs a Package given a list of scala files.
func newScalaPackage(
	args language.GenerateArgs,
	cfg *scalaconfig.Config,
	providerRegistry scalarule.ProviderRegistry,
	parser parser.Parser,
	universe resolver.Universe) *scalaPackage {
	s := &scalaPackage{
		args:             args,
		parser:           parser,
		universe:         universe,
		providerRegistry: providerRegistry,
		cfg:              cfg,
		rules:            make(map[string]*rule.Rule),
		resolveWork:      make([]func(), 0),
		ruleCoverage:     &packageRuleCoverage{},
	}
	s.gen = s.generateRules(true)
	// s.empty = s.generateRules(false)

	return s
}

// Config returns the the underlying config.
func (s *scalaPackage) Config() *scalaconfig.Config {
	return s.cfg
}

// ruleProvider returns the provider of a rule or nil if not known.
func (s *scalaPackage) ruleProvider(r *rule.Rule) scalarule.RuleProvider {
	if provider, ok := r.PrivateAttr(ruleProviderKey).(scalarule.RuleProvider); ok {
		return provider
	}
	return nil
}

func (s *scalaPackage) Resolve(
	c *config.Config,
	ix *resolve.RuleIndex,
	rc *repo.RemoteCache,
	r *rule.Rule,
	importsRaw interface{},
	from label.Label,
) {
	provider := s.ruleProvider(r)
	if provider == nil {
		log.Printf("no known rule provider for %v", from)
		return
	}
	fn := func() {
		provider.Resolve(&scalarule.ResolveContext{
			Config:    c,
			RuleIndex: ix,
			Rule:      r,
			From:      from,
			File:      s.args.File,
		}, importsRaw)
	}
	// the first resolve cycle populates the symbol scopes
	fn()
	s.resolveWork = append(s.resolveWork, fn)
}

// Finalize is called when all rules in the package have been resolved.
func (s *scalaPackage) Finalize() {
	for _, fn := range s.resolveWork {
		fn()
	}
}

// generateRules constructs a list of rules based on the configured set of rule
// configurations.
func (s *scalaPackage) generateRules(enabled bool) []scalarule.RuleProvider {
	rules := make([]scalarule.RuleProvider, 0)

	existingRulesByFQN := make(map[string][]*rule.Rule)
	if s.args.File != nil {
		for _, r := range s.args.File.Rules {
			fqn := fullyQualifiedLoadName(s.args.File.Loads, r.Kind())
			existingRulesByFQN[fqn] = append(existingRulesByFQN[fqn], r)
			if _, ok := s.providerRegistry.LookupProvider(fqn); ok {
				s.ruleCoverage.total += 1
			}
		}
	}

	configuredRules := s.cfg.ConfiguredRules()

	for _, ruleConfig := range configuredRules {
		if !ruleConfig.Enabled {
			// log.Printf("%s: skipping rule config %s (not enabled)", s.args.Rel, ruleConfig.Name)
			continue
		}
		rule := s.provideRule(ruleConfig)
		if rule != nil {
			rules = append(rules, rule)
		}
		existing := existingRulesByFQN[ruleConfig.Implementation]
		if len(existing) > 0 {
			for _, r := range existing {
				rule := s.resolveRule(ruleConfig, r)
				if rule != nil {
					s.ruleCoverage.managed += 1
					rules = append(rules, rule)
				}
			}
		}
		delete(existingRulesByFQN, ruleConfig.Implementation)
	}

	return rules
}

func (s *scalaPackage) provideRule(rc *scalarule.Config) scalarule.RuleProvider {
	provider, ok := s.providerRegistry.LookupProvider(rc.Implementation)
	if !ok {
		log.Fatalf(
			"%s: rule provider not registered: %q (available: %v)",
			s.args.Rel,
			rc.Implementation,
			s.providerRegistry.ProviderNames(),
		)
	}
	rc.Provider = provider

	return provider.ProvideRule(rc, s)
}

func (s *scalaPackage) resolveRule(rc *scalarule.Config, r *rule.Rule) scalarule.RuleProvider {
	provider, ok := s.providerRegistry.LookupProvider(rc.Implementation)
	if !ok {
		log.Fatalf(
			"%s: rule not registered: %q (available: %v)",
			s.args.Rel,
			rc.Implementation,
			s.providerRegistry.ProviderNames(),
		)
	}
	rc.Provider = provider

	if rr, ok := provider.(scalarule.RuleResolver); ok {
		return rr.ResolveRule(rc, s, r)
	}

	return nil
}

// GenerateArgs implements part of the scalarule.Package interface.
func (s *scalaPackage) GenerateArgs() language.GenerateArgs {
	return s.args
}

// GeneratedRules implements part of the scalarule.Package interface.
func (s *scalaPackage) GeneratedRules() (rules []*rule.Rule) {
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	return
}

// ParseRule implements part of the scalarule.Package interface.
func (s *scalaPackage) ParseRule(r *rule.Rule, attrName string) (scalarule.Rule, error) {

	dir := filepath.Join(s.repoRootDir(), s.args.Rel)
	srcs, err := glob.CollectFilenames(s.args.File, dir, r.Attr(attrName))
	if err != nil {
		return nil, err
	}
	scalaSrcs := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if !strings.HasSuffix(src, ".scala") {
			continue
		}
		scalaSrcs = append(scalaSrcs, src)
	}
	if len(scalaSrcs) == 0 {
		return nil, ErrRuleHasNoSrcs
	}

	from := s.cfg.MaybeRewrite(r.Kind(), label.Label{Pkg: s.args.Rel, Name: r.Name()})

	rule, err := s.parser.ParseScalaRule(r.Kind(), from, dir, scalaSrcs...)
	if err != nil {
		return nil, err
	}

	ctx := &scalaRuleContext{
		rule:        r,
		scalaConfig: s.cfg,
		resolver:    s.universe,
		scope:       s.universe,
	}

	return newScalaRule(ctx, rule), nil
}

// repoRootDir return the root directory of the repo.
func (s *scalaPackage) repoRootDir() string {
	return s.cfg.Config().RepoRoot
}

// Rules provides the aggregated rule list for the package.
func (s *scalaPackage) Rules() []*rule.Rule {
	return s.getProvidedRules(s.gen, true)
}

// Empty names the rules that can be deleted.
func (s *scalaPackage) Empty() []*rule.Rule {
	// it's a bit sad that we construct the full rules only for their kind and
	// name, but that's how it is right now.
	rules := s.getProvidedRules(s.empty, false)

	empty := make([]*rule.Rule, len(rules))
	for i, r := range rules {
		empty[i] = rule.NewRule(r.Kind(), r.Name())
	}

	return empty
}

func (s *scalaPackage) getProvidedRules(providers []scalarule.RuleProvider, shouldResolve bool) []*rule.Rule {
	rules := make([]*rule.Rule, 0)
	for _, p := range providers {
		r := p.Rule()
		if r == nil {
			continue
		}
		if shouldResolve {
			// record the association of the rule provider here for the resolver.
			r.SetPrivateAttr(ruleProviderKey, p)
			// log.Println("provided rule %s %s", r.Kind(), r.Name())
			s.rules[r.Name()] = r
		}
		rules = append(rules, r)
	}
	return rules
}
