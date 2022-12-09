package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ErrImportNotFound is an error value assigned to an Import when the import
// could not be resolved.
var ErrImportNotFound = fmt.Errorf("import not found")

// SymbolResolver knows how to resolve imports.
type SymbolResolver interface {
	// ResolveSymbol takes the given config, gazelle rule index, and an
	// import to resolve. Implementations should return ErrImportNotFound if
	// unsuccessful.  If multiple matches are found, return
	// ImportAmbiguousError.
	ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, sym string) (*Symbol, error)
}
