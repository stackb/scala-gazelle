package resolver

import (
	"fmt"
	"sort"
)

var globalConflictResolvers = make(globalConflictResolverMap)

// GlobalConflictResolverRegistry returns a default symbol provider registry.
// Third-party gazelle extensions can append to this list and configure their
// own implementations.
func GlobalConflictResolverRegistry() ConflictResolverRegistry {
	return globalConflictResolvers
}

type globalConflictResolverMap map[string]ConflictResolver

// GetConflictResolver implements part of the resolver.ConflictResolverRegistry
// interface.
func (r globalConflictResolverMap) GetConflictResolver(name string) (ConflictResolver, bool) {
	resolver, ok := r[name]
	return resolver, ok
}

// PutConflictResolver implements part of the resolver.ConflictResolverRegistry
// interface.
func (r globalConflictResolverMap) PutConflictResolver(name string, resolver ConflictResolver) error {
	if _, ok := r[name]; ok {
		return fmt.Errorf("duplicate ConflictResolver %q", name)
	}
	r[name] = resolver
	return nil
}

// GlobalConflictResolvers returns a sorted list of known conflict resolvers
func GlobalConflictResolvers() []ConflictResolver {
	keys := make([]string, 0, len(globalConflictResolvers))
	for k := range globalConflictResolvers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	resolvers := make([]ConflictResolver, len(keys))
	for i, k := range keys {
		resolvers[i] = globalConflictResolvers[k]
	}
	return resolvers
}
