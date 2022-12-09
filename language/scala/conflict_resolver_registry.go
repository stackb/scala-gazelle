package scala

import (
	"fmt"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// GetConflictResolver implements part of the resolver.ConflictResolverRegistry
// interface.
func (sl *scalaLang) GetConflictResolver(name string) (resolver.ConflictResolver, bool) {
	r, ok := sl.conflictResolvers[name]
	return r, ok
}

// PutConflictResolver implements part of the resolver.ConflictResolverRegistry
// interface.
func (sl *scalaLang) PutConflictResolver(name string, r resolver.ConflictResolver) error {
	if _, ok := sl.conflictResolvers[name]; ok {
		return fmt.Errorf("duplicate known rule: %s", name)
	}
	sl.conflictResolvers[name] = r
	return nil
}
