package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

func NewImportAmbiguousError(imp string, matches []resolve.FindResult) *ImportAmbiguousError {
	labels := make([]label.Label, len(matches))
	for i, match := range matches {
		labels[i] = match.Label
	}
	return &ImportAmbiguousError{
		Imp:    imp,
		Labels: labels,
	}
}

// ImportAmbiguousError is an error type assigned to an Import when multiple
// providers are possible for the given import.
type ImportAmbiguousError struct {
	// The Import that is ambiguous.
	Imp string
	// A list of possible providers for the import.
	Labels []label.Label
}

func (e *ImportAmbiguousError) Error() string {
	return fmt.Sprintf("found multiple matches for %q: %v", e.Imp, e.Labels)
}
