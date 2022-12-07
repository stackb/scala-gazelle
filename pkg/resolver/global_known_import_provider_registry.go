package resolver

import "fmt"

var globalKnownImportProviders = &globalKnownImportProviderRegistry{}

// GlobalKnownImportProviderRegistry returns a default import provider registry.
// Third-party gazelle extensions can append to this list and configure their
// own implementations.
func GlobalKnownImportProviderRegistry() KnownImportProviderRegistry {
	return globalKnownImportProviders
}

func GetNamedKnownImportProviders(names []string) (want []KnownImportProvider, err error) {
	all := GlobalKnownImportProviderRegistry().KnownImportProviders()
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
			return nil, fmt.Errorf("resolver.KnownImportProvider not found: %q", name)
		}
	}
	return
}

type globalKnownImportProviderRegistry struct {
	// knownImportProviders is a list of providers
	knownImportProviders []KnownImportProvider
}

// KnownImportProviders implements part of the
// resolver.KnownImportProviderRegistry interface.
func (r *globalKnownImportProviderRegistry) KnownImportProviders() []KnownImportProvider {
	return r.knownImportProviders
}

// AddKnownImportProvider implements part of the
// resolver.KnownImportProviderRegistry interface.
func (r *globalKnownImportProviderRegistry) AddKnownImportProvider(provider KnownImportProvider) error {
	for _, p := range r.knownImportProviders {
		if p.Name() == provider.Name() {
			return fmt.Errorf("duplicate resolver.KnownImportProvider %q", p.Name())
		}
	}
	r.knownImportProviders = append(r.knownImportProviders, provider)
	return nil
}
