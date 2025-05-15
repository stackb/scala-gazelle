package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/rs/zerolog"
)

func init() {
	cr := &PredefinedLabelConflictResolver{}
	GlobalConflictResolverRegistry().PutConflictResolver(cr.Name(), cr)
}

// PredefinedLabelConflictResolver implements a strategy where
type PredefinedLabelConflictResolver struct {
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *PredefinedLabelConflictResolver) Name() string {
	return "predefined_label"
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *PredefinedLabelConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config, logger zerolog.Logger) {
}

// CheckFlags implements part of the resolver.ConflictResolver interface.
func (s *PredefinedLabelConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// ResolveConflict implements part of the resolver.ConflictResolver interface.
// This implementation chooses symbols that have symbol.Label == label.NoLabel,
// which is the scenario when a symbol is provided by the platform / compiler,
// like "java.lang.String".
func (s *PredefinedLabelConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol) (*Symbol, bool) {
	if symbol.Label == label.NoLabel {
		return symbol, true
	}
	for _, sym := range symbol.Conflicts {
		if sym.Label == label.NoLabel {
			return sym, true
		}
	}
	return nil, false
}
