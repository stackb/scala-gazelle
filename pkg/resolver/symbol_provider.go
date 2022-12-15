package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// SymbolProvider is a flag-configurable entity that supplies symbols
// to a registry.
type SymbolProvider interface {
	// Providers have canonical names
	Name() string
	// RegisterFlags configures the flags.  RegisterFlags is called for all
	// providers whether they are enabled or not.
	RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config)
	// CheckFlags asserts that the flags are correct and provides a scope to
	// provide symbols to.  CheckFlags is only called if the provider is
	// enabled.
	CheckFlags(fs *flag.FlagSet, c *config.Config, scope Scope) error
	// OnResolve is a lifecycle hook that gets called when the resolve phase has
	// started.
	OnResolve() error
	// OnEnd is a lifecycle hook that gets called when the resolve phase has
	// ended.
	OnEnd() error
	// Providers typically manage a particular sub-space of labels.  For
	// example, the maven resolver may return true for labels like
	// "@maven//:junit_junit". The rule Index can be used to consult what type
	// of label from is, based on the rule characteristics.
	CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool
}
