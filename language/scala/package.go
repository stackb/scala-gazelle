package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/rules_proto/pkg/protoc"
)

const (
	ruleProviderKey = "_scala_rule_provider"
)

// scalaPackage provides a set of proto_library derived rules for the package.
type scalaPackage struct {
	// the registry to use
	ruleRegistry RuleRegistry
	// relative path of build file
	rel string
	// the config for this package
	cfg *scalaConfig
	// the list of '.scala' files
	files []*ScalaFile
	// the generated and empty rule providers
	gen, empty []RuleProvider
}

// newScalaPackage constructs a Package given a list of scala files.
func newScalaPackage(ruleRegistry RuleRegistry, rel string, cfg *scalaConfig, files ...*ScalaFile) *scalaPackage {
	s := &scalaPackage{
		ruleRegistry: ruleRegistry,
		rel:          rel,
		cfg:          cfg,
		files:        files,
	}
	s.gen = s.generateRules(true)
	s.empty = s.generateRules(false)
	return s
}

// generateRules constructs a list of rules based on the configured set of
// languages.
func (s *scalaPackage) generateRules(enabled bool) []RuleProvider {
	rules := make([]RuleProvider, 0)

	for _, rc := range s.cfg.configuredRules() {
		if enabled != rc.Enabled {
			continue
		}
		rules = append(rules, s.provideRule(rc))
	}

	return rules
}

func (s *scalaPackage) provideRule(rc *RuleConfig) RuleProvider {
	impl, err := globalRegistry.LookupRule(rc.Implementation)
	if err == ErrUnknownRule {
		log.Fatalf(
			"%s: rule not registered: %q (available: %v)",
			s.rel,
			rc.Implementation,
			globalRegistry.RuleNames(),
		)
	}
	rc.Impl = impl

	rule := impl.ProvideRule(rc, s.files)
	if rule == nil {
		return nil
	}

	return rule
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

			// imports := r.PrivateAttr(config.GazelleImportsKey)
			// if imports == nil {
			// 	lib := s.ruleLibs[p]
			// 	r.SetPrivateAttr(ProtoLibraryKey, lib)
			// }

			// NOTE: this is a bit of a hack: it would be preferable to populate
			// the global resolver with import specs during the .Imports()
			// function.  One would think that the RuleProvider could be set as
			// a PrivateAttr to be retrieved in the Imports() function. However,
			// the rule ref seems to have changed by that time, the PrivateAttr
			// is removed.  Maybe this is due to rule merges?  Very difficult to
			// track down bug that cost me days.
			from := label.New("", s.rel, r.Name())
			file := rule.EmptyFile("", s.rel)
			provideResolverImportSpecs(s.cfg.config, p, r, file, from)
		}

		rules = append(rules, r)
	}
	return rules
}

func provideResolverImportSpecs(c *config.Config, provider RuleProvider, r *rule.Rule, f *rule.File, from label.Label) {
	for _, imp := range provider.Imports(c, r, f) {
		protoc.GlobalResolver().Provide(
			"scala",
			imp.Lang,
			imp.Imp,
			from,
		)
	}
}
