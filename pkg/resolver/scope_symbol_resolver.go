package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ScopeSymbolResolver implements Resolver for symbols.
type ScopeSymbolResolver struct {
	scope Scope
}

func NewScopeSymbolResolver(scope Scope) *ScopeSymbolResolver {
	return &ScopeSymbolResolver{scope: scope}
}

// ResolveSymbol implements the ImportResolver interface
func (sr *ScopeSymbolResolver) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, symbol string) (*Symbol, bool) {
	if sym, ok := sr.scope.GetSymbol(symbol); ok {
		return sym, true
	}
	return nil, false
}
