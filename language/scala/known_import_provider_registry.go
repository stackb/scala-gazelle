package scala

import (
	"fmt"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// KnownImportProviders implements part of the
// resolver.KnownImportProviderRegistry interface.
func (sl *scalaLang) KnownImportProviders() []resolver.KnownImportProvider {
	return sl.knownImportProviders
}

// AddKnownImportProvider implements part of the
// resolver.KnownImportProviderRegistry interface.
func (sl *scalaLang) AddKnownImportProvider(provider resolver.KnownImportProvider) error {
	for _, p := range sl.knownImportProviders {
		if p.Name() == provider.Name() {
			return fmt.Errorf("duplicate resolver.KnownImportProvider %q", p.Name())
		}
	}
	sl.knownImportProviders = append(sl.knownImportProviders, provider)
	return nil
}
