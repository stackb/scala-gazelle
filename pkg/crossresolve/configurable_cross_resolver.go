package crossresolve

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// ConfigurableCrossResolver implementations support the CrossResolver interface
// as well as a subset of config.Configurer.  This interface is provided to
// support different implementations of a scala cross-resolver.  A simple
// implementation might be based off a CSV file, whereas a larger monorepo may
// require a more sophisticated cache.
type ConfigurableCrossResolver interface {
	resolve.CrossResolver
	// RegisterFlags implements part of the config.Configurer interface.
	RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config)
	// CheckFlags implements part of the config.Configurer interface.
	CheckFlags(fs *flag.FlagSet, c *config.Config) error
}

// GazellePhaseTransitionListener is an optional interface for a cross-resolver
// that wants phase transition notification.
type GazellePhaseTransitionListener interface {
	OnResolve()
	OnEnd()
}

// LabelOwner is an optional interface for a cross-resolver
// that can claims a particular sub-space of labels.  For example, the
// maven resolver may return true for labels like "@maven//:junit_junit".
// the ruleIndex can be used to consult what type of label from is, based
// on the rule characteristics.  If no rule corresponding to the given
// label is found, ruleIndex returns nil, false.
type LabelOwner interface {
	IsOwner(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool
}

func crossResolverNameMatches(resolverLang, lang string, imp resolve.ImportSpec) bool {
	return lang == resolverLang || imp.Lang == resolverLang
}
