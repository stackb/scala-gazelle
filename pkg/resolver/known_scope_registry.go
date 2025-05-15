package resolver

// KnownScopeRegistry is an index of scopes keyed by their filename.  For
// example, if one needed to resolve the symbol 'A' within file
// `a/b/c/Main.scala`, the registry could be used to gain the scope to perform
// this lookup.
type KnownScopeRegistry interface {
	// GetScope does a lookup of the given label and returns the
	// known rule.  If not known `(nil, false)` is returned.
	GetKnownScope(name string) (Scope, bool)

	// PutFileScope adds the given known rule to the registry.  It is an
	// error to attempt duplicate registration of the same rule twice.
	// Implementations should use the google.golang.org/grpc/status.Errorf for
	// error types.
	PutKnownScope(name string, scope Scope) error
}
