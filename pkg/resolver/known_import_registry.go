package resolver

type KnownImportRegistry interface {
	// RequireImport does a lookup of the given import symbol and returns the
	// known import.  If not known `(nil, false)` is returned.
	RequireImport(imp string) (*KnownImport, bool)

	// ProvideImport adds the given known import to the registry.
	ProvideImport(known *KnownImport) error
}
