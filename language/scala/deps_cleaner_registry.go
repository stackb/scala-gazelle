package scala

import (
	"fmt"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// GetDepsCleaner implements part of the resolver.DepsCleanerRegistry
// interface.
func (sl *scalaLang) GetDepsCleaner(name string) (resolver.DepsCleaner, bool) {
	r, ok := sl.depsCleaners[name]
	return r, ok
}

// PutDepsCleaner implements part of the resolver.DepsCleanerRegistry
// interface.
func (sl *scalaLang) PutDepsCleaner(name string, r resolver.DepsCleaner) error {
	if _, ok := sl.depsCleaners[name]; ok {
		return fmt.Errorf("duplicate conflict resolver: %s", name)
	}
	sl.depsCleaners[name] = r
	return nil
}
