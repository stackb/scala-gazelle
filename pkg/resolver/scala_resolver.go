package resolver

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ScalaResolver implements KnownImportResolver for scala imports.  Patterns
// like '_root_.' or "._" are stripped.
type ScalaResolver struct {
	next KnownImportResolver
}

// NewScalaResolver constructs a new resolver that chains to the given resolver.
func NewScalaResolver(next KnownImportResolver) *ScalaResolver {
	return &ScalaResolver{next}
}

// ResolveKnownImport implements the KnownImportResolver interface
func (sr *ScalaResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	imp = strings.TrimPrefix(imp, "_root_.")
	imp = strings.TrimSuffix(imp, "._")

	// TODO: if have unresolved dep, try add 'scala.'

	return sr.next.ResolveKnownImport(c, ix, from, lang, imp)
}
