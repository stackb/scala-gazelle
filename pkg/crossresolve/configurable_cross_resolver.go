package crossresolve

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
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
// that wants phase transition notification.  Errors are considered fatal.
type GazellePhaseTransitionListener interface {
	OnResolve()
	OnEnd()
}
