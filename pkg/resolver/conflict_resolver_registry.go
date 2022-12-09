package resolver

// ConflictResolverRegistry is an index of known conflict resolvers keyed by their name.
type ConflictResolverRegistry interface {
	// GetConflictResolver returns the named resolver.  If not known `(nil,
	// false)` is returned.
	GetConflictResolver(name string) (ConflictResolver, bool)

	// PutConflictResolver adds the given known rule to the registry.  It is an
	// error to attempt duplicate registration of the same rule twice.
	// Implementations should use the google.golang.org/grpc/status.Errorf for
	// error types.
	PutConflictResolver(name string, r ConflictResolver) error
}
