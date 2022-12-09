package resolver

// KnownImportRegistry is an index of all imports that are known to the
// extension at the beginning of the deps resolution phase.
type KnownImportRegistry interface {
	// GetKnownImport does a lookup of the given import symbol and returns the
	// known import.  If not known `(nil, false)` is returned.
	GetKnownImport(imp string) (*KnownImport, bool)

	// GetKnownImports does a lookup of the given prefix and returns the
	// known imports.
	GetKnownImports(prefix string) []*KnownImport

	// PutKnownImport adds the given known import to the registry.  It is an
	// error to attempt duplicate registration of the same import twice.
	PutKnownImport(known *KnownImport) error
}
