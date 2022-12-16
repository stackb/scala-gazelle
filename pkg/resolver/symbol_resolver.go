package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ErrSymbolNotFound is an error value assigned to an Import when the name could
// not be resolved.
var ErrSymbolNotFound = fmt.Errorf("symbol not found")

// SymbolResolver knows how to resolve imports.
type SymbolResolver interface {
	// ResolveSymbol takes the given config, gazelle rule index, and an
	// import to resolve.
	ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, sym string) (*Symbol, error)
}
