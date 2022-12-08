package scalarule

// ProviderRegistry represents a library of rule provider implementations.
type ProviderRegistry interface {
	// ProviderNames returns a sorted list of rule names.
	ProviderNames() []string
	// LookupProvider returns the implementation under the given name.  If the
	// rule is not found, false is returned.
	LookupProvider(name string) (Provider, bool)
	// RegisterProvider installs a Provider implementation under the given name
	// in the global rule registry.  Error will occur if the same rule is
	// registered multiple times.
	RegisterProvider(name string, provider Provider) error
}
