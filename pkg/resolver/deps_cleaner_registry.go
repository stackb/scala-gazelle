package resolver

// DepsCleanerRegistry is an index of known conflict resolvers keyed by their name.
type DepsCleanerRegistry interface {
	// GetDepsCleaner returns the named resolver.  If not known `(nil,
	// false)` is returned.
	GetDepsCleaner(name string) (DepsCleaner, bool)

	// PutDepsCleaner adds the given known rule to the registry.  It is an
	// error to attempt duplicate registration of the same rule twice.
	// Implementations should use the google.golang.org/grpc/status.Errorf for
	// error types.
	PutDepsCleaner(name string, r DepsCleaner) error
}
