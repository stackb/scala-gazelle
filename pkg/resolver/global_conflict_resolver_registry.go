package resolver

import "fmt"

var globalConflictResolverMap = &globalConflictResolverRegistry{}

// GlobalConflictResolverRegistry returns a default symbol provider registry.
// Third-party gazelle extensions can append to this list and configure their
// own implementations.
func GlobalConflictResolverRegistry() ConflictResolverRegistry {
	return globalConflictResolverMap
}

func GetNamedConflictResolvers(names []string) (want []ConflictResolver, err error) {
	for _, name := range names {
		resolver, ok := globalConflictResolverMap.GetConflictResolver(name)
		if !ok {
			return nil, fmt.Errorf("ConflictResolver not found: %q", name)
		}
		want = append(want, resolver)
	}
	return
}

type globalConflictResolverRegistry struct {
	resolvers map[string]ConflictResolver
}

// GetConflictResolver implements part of the resolver.ConflictResolverRegistry
// interface.
func (r *globalConflictResolverRegistry) GetConflictResolver(name string) (ConflictResolver, bool) {
	resolver, ok := r.resolvers[name]
	return resolver, ok
}

// PutConflictResolver implements part of the resolver.ConflictResolverRegistry
// interface.
func (r *globalConflictResolverRegistry) PutConflictResolver(name string, resolver ConflictResolver) error {
	if _, ok := r.resolvers[name]; ok {
		return fmt.Errorf("duplicate ConflictResolver %q", name)
	}
	r.resolvers[name] = resolver
	return nil
}
