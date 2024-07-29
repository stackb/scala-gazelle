package resolver

import (
	"fmt"
	"sort"
)

var globalDepsCleaners = make(globalDepsCleanerMap)

// GlobalDepsCleanerRegistry returns a default deps cleaner registry.
// Third-party gazelle extensions can append to this list and configure their
// own implementations.
func GlobalDepsCleanerRegistry() DepsCleanerRegistry {
	return globalDepsCleaners
}

type globalDepsCleanerMap map[string]DepsCleaner

// GetDepsCleaner implements part of the resolver.DepsCleanerRegistry
// interface.
func (r globalDepsCleanerMap) GetDepsCleaner(name string) (DepsCleaner, bool) {
	resolver, ok := r[name]
	return resolver, ok
}

// PutDepsCleaner implements part of the resolver.DepsCleanerRegistry
// interface.
func (r globalDepsCleanerMap) PutDepsCleaner(name string, resolver DepsCleaner) error {
	if _, ok := r[name]; ok {
		return fmt.Errorf("duplicate DepsCleaner %q", name)
	}
	r[name] = resolver
	return nil
}

// GlobalDepsCleaners returns a sorted list of known conflict resolvers
func GlobalDepsCleaners() []DepsCleaner {
	keys := make([]string, 0, len(globalDepsCleaners))
	for k := range globalDepsCleaners {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	resolvers := make([]DepsCleaner, len(keys))
	for i, k := range keys {
		resolvers[i] = globalDepsCleaners[k]
	}
	return resolvers
}
