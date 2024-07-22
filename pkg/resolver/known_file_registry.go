package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KnownFileRegistry is an index of known files keyed by their package name.
type KnownFileRegistry interface {
	// GetKnownFile does a lookup of the given label and returns the
	// known file.  If not known `(nil, false)` is returned.
	GetKnownFile(pkg string) (*rule.File, bool)

	// PutKnownFile adds the given known rule to the registry.  It is an
	// error to attempt duplicate registration of the same file twice.
	PutKnownFile(pkg string, r *rule.File) error
}
