package scala

import "github.com/stackb/scala-gazelle/pkg/resolver"

// SymbolProviders implements part of the
// resolver.SymbolProviderRegistry interface.
func (sl *scalaLang) SymbolProviders() []resolver.SymbolProvider {
	return resolver.GlobalSymbolProviderRegistry().SymbolProviders()
}

// AddSymbolProvider implements part of the
// resolver.SymbolProviderRegistry interface.
func (sl *scalaLang) AddSymbolProvider(provider resolver.SymbolProvider) error {
	sl.logger.Debug().Msgf("adding symbol provider: %s", provider.Name())
	return resolver.GlobalSymbolProviderRegistry().AddSymbolProvider(provider)
}
