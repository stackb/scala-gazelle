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
)

const (
	ruleProviderKey = "_scala_rule_provider"
)

type ScalaPackage interface {
	// ParseScalaRule parses the given rule from its 'srcs' attribute.
	ParseScalaRule(r *rule.Rule) (*ScalaRule, error)
}

// scalaPackage provides a set of proto_library derived rules for the package.
type scalaPackage struct {
	// rel is the package (args.Rel)
	rel string
	// parser is the file parser
	parser scalaparse.Parser
	// importResolver is the parent importResolver
	importResolver resolver.KnownImportResolver
	// the registry to use
	ruleRegistry RuleRegistry
	// the build file
	file *rule.File
	// the config for this package
	cfg *scalaConfig
	// the generated and empty rule providers
	gen, empty []RuleProvider
	// resolved is the final state of generated rules, by name.
	rules map[string]*rule.Rule
}

// newScalaPackage constructs a Package given a list of scala files.
func newScalaPackage(rel string, file *rule.File, cfg *scalaConfig, ruleRegistry RuleRegistry, parser scalaparse.Parser, importResolver resolver.KnownImportResolver) *scalaPackage {
	s := &scalaPackage{
		rel:            rel,
		parser:         parser,
		importResolver: importResolver,
		ruleRegistry:   ruleRegistry,
		file:           file,
		cfg:            cfg,
		rules:          make(map[string]*rule.Rule),
	}
	s.gen = s.generateRules(true)
	// s.empty = s.generateRules(false)

	return s
}

// Config returns the the underlying config.
func (s *scalaPackage) Config() *scalaConfig {
	return s.cfg
}

// getRule returns the named rule, if it exists
func (s *scalaPackage) getRule(name string) (*rule.Rule, bool) {
	got, ok := s.rules[name]
	return got, ok
}

// ruleProvider returns the provider of a rule or nil if not known.
func (s *scalaPackage) ruleProvider(r *rule.Rule) RuleProvider {
	if provider, ok := r.PrivateAttr(ruleProviderKey).(RuleProvider); ok {
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
func (s *scalaPackage) generateRules(enabled bool) []RuleProvider {
	rules := make([]RuleProvider, 0)

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

func (s *scalaPackage) provideRule(rc *RuleConfig) RuleProvider {
	impl, err := s.ruleRegistry.LookupRule(rc.Implementation)
	if err == ErrUnknownRule {
		log.Fatalf(
			"%s: rule not registered: %q (available: %v)",
			s.rel,
			rc.Implementation,
			s.ruleRegistry.RuleNames(),
		)
	}
	rc.Impl = impl

	return impl.ProvideRule(rc, s)
}

func (s *scalaPackage) resolveRule(rc *RuleConfig, r *rule.Rule) RuleProvider {
	impl, err := s.ruleRegistry.LookupRule(rc.Implementation)
	if err == ErrUnknownRule {
		log.Fatalf(
			"%s: rule not registered: %q (available: %v)",
			s.rel,
			rc.Implementation,
			globalRuleRegistry.RuleNames(),
		)
	}
	rc.Impl = impl

	if rr, ok := impl.(RuleResolver); ok {
		return rr.ResolveRule(rc, s, r)
	}

	return nil
}

// ParseScalaRule implements part of the ScalaPackage interface.
func (s *scalaPackage) ParseScalaRule(r *rule.Rule) (*ScalaRule, error) {
	filenames, err := glob.CollectFilenames(s.file, s.repoRootDir(), s.rel, r.Attr("srcs"))
	if err != nil {
		return nil, err
	}

	if len(filenames) == 0 {
		return nil, err
	}

	from := label.New("", s.rel, r.Name())

	files, err := parseScalaFiles(s.repoRootDir(), from, r.Kind(), filenames, s.parser)
	if err != nil {
		return nil, err
	}

	return NewScalaRule(s.importResolver, r, from, files), nil
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

func (s *scalaPackage) getProvidedRules(providers []RuleProvider, shouldResolve bool) []*rule.Rule {
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
