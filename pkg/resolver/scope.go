package resolver

// Scope is an index of known symbols.
type Scope interface {
	// GetSymbol does a lookup of the given import symbol and returns the
	// known import.  If not known `(nil, false)` is returned.
	GetSymbol(imp string) (*Symbol, bool)

	// GetSymbols does a lookup of the given prefix and returns the
	// symbols.
	GetSymbols(prefix string) []*Symbol

	// PutSymbol adds the given known import to the registry.  It is an
	// error to attempt duplicate registration of the same import twice.
	PutSymbol(known *Symbol) error
}
