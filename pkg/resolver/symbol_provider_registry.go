package resolver

// SymbolProviderRegistry is an index of import providers.
type SymbolProviderRegistry interface {
	// SymbolProviders returns a list of all known providers.
	SymbolProviders() []SymbolProvider

	// AddSymbolProvider adds the given known import provider to the
	// registry.  It is an error to add the same namedprovider twice.
	AddSymbolProvider(provider SymbolProvider) error
}
