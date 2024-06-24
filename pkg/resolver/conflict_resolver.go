package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// ConflictResolver implementations are capable of applying a conflict
// resolution strategy for conflicting resolved import symbols.
type ConflictResolver interface {
	// Name is the canonical name for the resolver
	Name() string
	// RegisterFlags configures the flags.  RegisterFlags is called for all
	// resolvers whether they are enabled or not.
	RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config)
	// CheckFlags asserts that the flags are correct.  CheckFlags is only called
	// if the resolver is enabled.
	CheckFlags(fs *flag.FlagSet, c *config.Config) error
	// ResolveConflict takes the context rule and imports, and the target symbol
	// with conflicts to resolve. The ImportMap MAY be modified during the
	// operation.  The function MAY return (nil, true) in which case the symbol
	// should be elided from further processing.
	ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol, from label.Label) (*Symbol, bool)
}
