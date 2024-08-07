package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// DepsCleaner implementations are capable of applying some sort of cleanup
// strategy on the post-resolved deps of a rule.
type DepsCleaner interface {
	// Name is the canonical name for the resolver
	Name() string
	// RegisterFlags configures the flags.  RegisterFlags is called for all
	// resolvers whether they are enabled or not.
	RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config)
	// CheckFlags asserts that the flags are correct.  CheckFlags is only called
	// if the resolver is enabled.
	CheckFlags(fs *flag.FlagSet, c *config.Config) error
	// CleanDeps takes the context rule and a map of labels that represent the
	// incoming deps.  The cleaner implementation should assign the value under
	// the dep label as false if the dep is not wanted.
	CleanDeps(deps map[label.Label]bool, r *rule.Rule, from label.Label)
}
