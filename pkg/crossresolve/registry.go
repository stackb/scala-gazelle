package crossresolve

import (
	"errors"
	"log"
	"sort"
)

// ErrUnknownResolver is the error returned when a resolver is not known.
var ErrUnknownResolver = errors.New("unknown cross resolver")

// Registry represents a library of crossresolver implementations.
type Registry interface {
	// ResolverNames returns a sorted list of resolver names.
	ResolverNames() []string
	// LookupResolver returns the implementation under the given name.  If the resolver
	// is not found, ErrUnknownResolver is returned.
	LookupResolver(name string) (ConfigurableCrossResolver, error)
	// MustRegisterResolver installs a ConfigurableCrossResolver implementation under the given
	// name in the global resolver registry.  Panic will occur if the same resolver is
	// registered multiple times.
	MustRegisterResolver(name string, resolver ConfigurableCrossResolver) Registry
}

// Resolvers returns a reference to the global Registry
func Resolvers() Registry {
	return globalRegistry
}

// registry is the default registry singleton.
var globalRegistry = &registry{
	resolvers: make(map[string]ConfigurableCrossResolver),
}

// registry implements Registry.
type registry struct {
	resolvers map[string]ConfigurableCrossResolver
}

// ResolverNames implements part of the Registry interface.
func (p *registry) ResolverNames() []string {
	names := make([]string, 0)
	for name := range p.resolvers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MustRegisterResolver implements part of the Registry interface.
func (p *registry) MustRegisterResolver(name string, resolver ConfigurableCrossResolver) Registry {
	_, ok := p.resolvers[name]
	if ok {
		panic("duplicate CrossResolver registration: " + name)
	}
	p.resolvers[name] = resolver
	log.Println("registered resolver:", name)
	return p
}

// LookupResolver implements part of the Registry interface.
func (p *registry) LookupResolver(name string) (ConfigurableCrossResolver, error) {
	rule, ok := p.resolvers[name]
	if !ok {
		return nil, ErrUnknownResolver
	}
	return rule, nil
}
