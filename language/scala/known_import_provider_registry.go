package scala

import (
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// KnownImportProviders implements part of the
// resolver.KnownImportProviderRegistry interface.
func (sl *scalaLang) KnownImportProviders() []resolver.KnownImportProvider {
	return resolver.GlobalKnownImportProviderRegistry().KnownImportProviders()
}

// AddKnownImportProvider implements part of the
// resolver.KnownImportProviderRegistry interface.
func (sl *scalaLang) AddKnownImportProvider(provider resolver.KnownImportProvider) error {
	return resolver.GlobalKnownImportProviderRegistry().AddKnownImportProvider(provider)
}
