package resolver

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func NewPreferredDepsConflictResolver(name string, preferred map[string]label.Label) *PreferredDepsConflictResolver {
	return &PreferredDepsConflictResolver{name, preferred}
}

// PreferredDepsConflictResolver implements a strategy where
type PreferredDepsConflictResolver struct {
	name      string
	preferred map[string]label.Label
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *PreferredDepsConflictResolver) Name() string {
	return s.name
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *PreferredDepsConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.ConflictResolver interface.
func (s *PreferredDepsConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// ResolveConflict implements part of the resolver.ConflictResolver interface.
// This implementation uses the preferred package map to try and match the
// symbol.Name against the preferred package key.  If found, the matching dep
// Label is used to select the correct one from the conflict list.
func (s *PreferredDepsConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol) (*Symbol, bool) {
	if want, ok := s.preferred[symbol.Name]; ok {
		if symbol.Label == want {
			return symbol, true
		}
		for _, sym := range symbol.Conflicts {
			if sym.Label == want {
				return sym, true
			}
		}
	}
	return nil, false
}
