package crossresolve

import (
	"errors"
	"log"
)

// ErrUnknownResolver is the error returned when a resolver is not known.
var ErrUnknownResolver = errors.New("unknown cross resolver")

// Registry represents a library of crossresolver implementations.
type Registry interface {
	// ByName returns a map of resolver implementations keyed by their name.
	ByName() map[string]ConfigurableCrossResolver
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

// ByName implements part of the Registry interface.
func (p *registry) ByName() map[string]ConfigurableCrossResolver {
	return p.resolvers
}

// MustRegisterResolver implements part of the Registry interface.
func (p *registry) MustRegisterResolver(name string, resolver ConfigurableCrossResolver) Registry {
	_, ok := p.resolvers[name]
	if ok {
		log.Println("warning: duplicate CrossResolver registration: " + name)
	}
	p.resolvers[name] = resolver
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
