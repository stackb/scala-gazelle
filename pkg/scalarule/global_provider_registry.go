package scalarule

import (
	"fmt"
	"sort"
)

// globalProviderRegistry is the default registry singleton.
var globalProviderRegistry = NewProviderRegistryMap()

// GlobalProviderRegistry returns a reference to the global ProviderRegistry
// implementation.
func GlobalProviderRegistry() ProviderRegistry {
	return globalProviderRegistry
}

// ProviderRegistryMap implements ProviderRegistry using a map.
type ProviderRegistryMap struct {
	providers map[string]Provider
}

func NewProviderRegistryMap() *ProviderRegistryMap {
	return &ProviderRegistryMap{
		providers: make(map[string]Provider),
	}
}

// ProviderNames implements part of the ProviderRegistry interface.
func (p *ProviderRegistryMap) ProviderNames() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterProvider implements part of the ProviderRegistry interface.
func (p *ProviderRegistryMap) RegisterProvider(name string, provider Provider) error {
	_, ok := p.providers[name]
	if ok {
		return fmt.Errorf("duplicate rule provider registration: %q", name)
	}
	p.providers[name] = provider
	return nil
}

// LookupProvider implements part of the RuleRegistry interface.
func (p *ProviderRegistryMap) LookupProvider(name string) (Provider, bool) {
	provider, ok := p.providers[name]
	if !ok {
		return nil, false
	}
	return provider, true
}
