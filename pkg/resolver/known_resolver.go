package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// KnownResolver implements Resolver for known imports.
type KnownResolver struct {
	// known is the known import registry
	known KnownImportRegistry
}

func NewKnownResolver(known KnownImportRegistry) *KnownResolver {
	return &KnownResolver{known: known}
}

// ResolveKnownImport implements the ImportResolver interface
func (sr *KnownResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, symbol string) (*KnownImport, error) {
	if known, ok := sr.known.GetKnownImport(symbol); ok {
		return known, nil
	}
	return nil, ErrImportNotFound
}
