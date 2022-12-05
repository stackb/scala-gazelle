package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ChainResolver implements KnownImportResolver over a chain of resolvers.
type ChainResolver struct {
	chain []KnownImportResolver
}

func NewChainResolver(chain ...KnownImportResolver) *ChainResolver {
	return &ChainResolver{
		chain: chain,
	}
}

// ResolveKnownImport implements the KnownImportResolver interface
func (r *ChainResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	for _, next := range r.chain {
		known, err := next.ResolveKnownImport(c, ix, from, lang, imp)
		if err == nil {
			return known, err
		}
		if err == ErrImportNotFound {
			continue
		}
		return nil, err
	}

	return nil, ErrImportNotFound
}
