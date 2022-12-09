package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/glob"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaparse"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	ruleProviderKey = "_scala_rule_provider"
)

// scalaPackage provides a set of proto_library derived rules for the package.
type scalaPackage struct {
	// rel is the package (args.Rel)
	rel string
	// parser is the file parser
	parser scalaparse.Parser
	// importResolver is the parent importResolver
	importResolver resolver.Universe
	// the registry to use
	providerRegistry scalarule.ProviderRegistry
	// the build file
	file *rule.File
	// the config for this package
	cfg *scalaConfig
	// the generated and empty rule providers
	gen, empty []scalarule.RuleProvider
	// resolved is the final state of generated rules, by name.
	rules map[string]*rule.Rule
}

// newScalaPackage constructs a Package given a list of scala files.
func newScalaPackage(rel string, file *rule.File, cfg *scalaConfig, providerRegistry scalarule.ProviderRegistry, parser scalaparse.Parser, importResolver resolver.Universe) *scalaPackage {
	s := &scalaPackage{
		rel:              rel,
		parser:           parser,
		importResolver:   importResolver,
		providerRegistry: providerRegistry,
		file:             file,
		cfg:              cfg,
		rules:            make(map[string]*rule.Rule),
	}
	s.gen = s.generateRules(true)
	// s.empty = s.generateRules(false)

	return s
}

// Config returns the the underlying config.
func (s *scalaPackage) Config() *scalaConfig {
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
	provider.Resolve(c, ix, r, importsRaw, from)
	s.rules[r.Name()] = r // TODO(pcj): do we need this assignment here?
}

// generateRules constructs a list of rules based on the configured set of rule
// configurations.
func (s *scalaPackage) generateRules(enabled bool) []scalarule.RuleProvider {
	rules := make([]scalarule.RuleProvider, 0)

	existingRulesByFQN := make(map[string][]*rule.Rule)
	if s.file != nil {
		for _, r := range s.file.Rules {
			fqn := fullyQualifiedLoadName(s.file.Loads, r.Kind())
			existingRulesByFQN[fqn] = append(existingRulesByFQN[fqn], r)
		}
	}

	configuredRules := s.cfg.configuredRules()

	for _, rc := range configuredRules {
		// if enabled != rc.Enabled {
		if !rc.Enabled {
			// log.Printf("%s: skipping rule config %s (not enabled)", s.rel, rc.Name)
			continue
		}
		rule := s.provideRule(rc)
		if rule != nil {
			rules = append(rules, rule)
		}
		existing := existingRulesByFQN[rc.Implementation]
		if len(existing) > 0 {
			for _, r := range existing {
				rule := s.resolveRule(rc, r)
				if rule != nil {
					rules = append(rules, rule)
				}
			}
		}
		delete(existingRulesByFQN, rc.Implementation)
	}

	return rules
}

func (s *scalaPackage) provideRule(rc *scalarule.Config) scalarule.RuleProvider {
	provider, ok := s.providerRegistry.LookupProvider(rc.Implementation)
	if !ok {
		log.Fatalf(
			"%s: rule provider not registered: %q (available: %v)",
			s.rel,
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
			s.rel,
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

// ParseRule implements part of the scalarule.Package interface.
func (s *scalaPackage) ParseRule(r *rule.Rule, attrName string) (scalarule.Rule, error) {
	filenames, err := glob.CollectFilenames(s.file, s.repoRootDir(), s.rel, r.Attr(attrName))
	if err != nil {
		return nil, err
	}

	from := label.New("", s.rel, r.Name())

	files, err := parseScalaFiles(s.repoRootDir(), from, r.Kind(), filenames, s.parser)
	if err != nil {
		return nil, err
	}

	return newScalaRule(
		s.cfg,
		s.importResolver,
		s.importResolver,
		resolver.NewTrieScope(),
		r, from, files,
	), nil
}

// repoRootDir return the root directory of the repo.
func (s *scalaPackage) repoRootDir() string {
	return s.cfg.config.RepoRoot
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

func parseScalaFiles(dir string, from label.Label, kind string, srcs []string, parser scalaparse.Parser) ([]*sppb.File, error) {
	if index, err := parser.ParseScalaFiles(from, kind, dir, srcs...); err != nil {
		return nil, err
	} else {
		return index.Files, nil
	}
}
