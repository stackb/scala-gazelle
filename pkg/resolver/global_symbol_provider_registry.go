package resolver

import "fmt"

var globalSymbolProviders = &globalSymbolProviderRegistry{}

// GlobalSymbolProviderRegistry returns a default symbol provider registry.
// Third-party gazelle extensions can append to this list and configure their
// own implementations.
func GlobalSymbolProviderRegistry() SymbolProviderRegistry {
	return globalSymbolProviders
}

func GetNamedSymbolProviders(names []string) (want []SymbolProvider, err error) {
	all := GlobalSymbolProviderRegistry().SymbolProviders()
	for _, name := range names {
		found := false
		for _, provider := range all {
			if name == provider.Name() {
				want = append(want, provider)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("resolver.SymbolProvider not found: %q", name)
		}
	}
	return
}

type globalSymbolProviderRegistry struct {
	providers []SymbolProvider
}

// SymbolProviders implements part of the
// resolver.SymbolProviderRegistry interface.
func (r *globalSymbolProviderRegistry) SymbolProviders() []SymbolProvider {
	return r.providers
}

// AddSymbolProvider implements part of the
// resolver.SymbolProviderRegistry interface.
func (r *globalSymbolProviderRegistry) AddSymbolProvider(provider SymbolProvider) error {
	for _, p := range r.providers {
		if p.Name() == provider.Name() {
			return fmt.Errorf("duplicate resolver.SymbolProvider %q", p.Name())
		}
	}
	r.providers = append(r.providers, provider)
	return nil
}
