package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

func NewAmbiguousSymbolError(name string, matches []resolve.FindResult) *AmbiguousSymbolError {
	labels := make([]label.Label, len(matches))
	for i, match := range matches {
		labels[i] = match.Label
	}
	return &AmbiguousSymbolError{
		Name:   name,
		Labels: labels,
	}
}

// AmbiguousSymbolError is an error type assigned to an Import when multiple
// providers are possible for the given import.
type AmbiguousSymbolError struct {
	// The Import that is ambiguous.
	Name string
	// A list of possible providers for the import.
	Labels []label.Label
}

func (e *AmbiguousSymbolError) Error() string {
	return fmt.Sprintf("found multiple matches for %q: %v", e.Name, e.Labels)
}
