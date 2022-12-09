package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// ConflictResolver implementations are capable of applying a conflict
// resolution strategy for conflicting resolved import symbols.
type ConflictResolver interface {
	// RegisterFlags configures the flags.  RegisterFlags is called for all
	// resolvers whether they are enabled or not.
	RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config)
	// CheckFlags asserts that the flags are correct.  CheckFlags is only called
	// if the resolver is enabled.
	CheckFlags(fs *flag.FlagSet, c *config.Config, scope Scope) error
	// ResolveConflict takes the context rule and imports, and the target symbol
	// with conflicts to resolve.
	ResolveConflict(r *rule.Rule, imports ImportMap, symbol *Symbol) (*Symbol, bool)
}
