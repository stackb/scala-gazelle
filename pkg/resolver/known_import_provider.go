package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KnownImportProvider is a flag-configurable entity that supplies known imports
// to a registry.
type KnownImportProvider interface {
	// Providers have canonical names
	Name() string
	// RegisterFlags configures the flags.
	RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config)
	// CheckFlags asserts that the flags are correct.
	CheckFlags(fs *flag.FlagSet, c *config.Config, registry KnownImportRegistry) error
	// Providers typically manage a particular sub-space of labels.  For example,
	// the maven resolver may return true for labels like
	// "@maven//:junit_junit". The rule Index can be used to consult what type
	// of label from is, based on the rule characteristics.
	CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool
	// OnResolve is a lifecycle hook that gets called when the resolve phase is
	// beginning.
	OnResolve()
}
