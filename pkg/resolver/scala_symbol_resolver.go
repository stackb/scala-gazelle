package resolver

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ScalaSymbolResolver implements SymbolResolver for scala imports.  Patterns
// like '_root_.' or "._" are stripped.
type ScalaSymbolResolver struct {
	next SymbolResolver
}

// NewScalaSymbolResolver constructs a new resolver that chains to the given resolver.
func NewScalaSymbolResolver(next SymbolResolver) *ScalaSymbolResolver {
	return &ScalaSymbolResolver{next}
}

// ResolveSymbol implements the SymbolResolver interface
func (sr *ScalaSymbolResolver) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*Symbol, error) {
	imp = strings.TrimPrefix(imp, "_root_.")
	imp = strings.TrimSuffix(imp, "._")

	// TODO: if have unresolved dep, try add 'scala.'

	return sr.next.ResolveSymbol(c, ix, from, lang, imp)
}
