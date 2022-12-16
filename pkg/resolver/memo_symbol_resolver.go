package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// MemoSymbolResolver implements SymbolResolver, memoizing results.
type MemoSymbolResolver struct {
	next  SymbolResolver
	cache map[string]*Symbol
}

func NewMemoSymbolResolver(next SymbolResolver) *MemoSymbolResolver {
	return &MemoSymbolResolver{
		next:  next,
		cache: make(map[string]*Symbol),
	}
}

// ResolveSymbol implements the SymbolResolver interface
func (r *MemoSymbolResolver) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*Symbol, bool) {
	if sym, ok := r.cache[imp]; ok {
		return sym, true
	}
	if sym, ok := r.next.ResolveSymbol(c, ix, from, lang, imp); ok {
		r.cache[imp] = sym
		return sym, true
	}
	return nil, false
}
