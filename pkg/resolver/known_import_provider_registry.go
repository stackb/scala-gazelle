package resolver

// KnownImportProviderRegistry is an index of import providers.
type KnownImportProviderRegistry interface {
	// KnownImportProviders returns a list of all known providers.
	KnownImportProviders() []KnownImportProvider

	// AddKnownImportProvider adds the given known import provider to the
	// registry.  It is an error to add the same namedprovider twice.
	AddKnownImportProvider(provider KnownImportProvider) error
}
