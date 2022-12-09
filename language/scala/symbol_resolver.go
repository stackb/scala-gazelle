package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// newSymbolResolver constructs the top-level known import resolver.
func newSymbolResolver(scope resolver.Scope) resolver.SymbolResolver {
	chain := resolver.NewChainSymbolResolver(
		// override resolver is the least performant!
		resolver.NewMemoSymbolResolver(resolver.NewOverrideSymbolResolver(scalaLangName)),
		resolver.NewScopeSymbolResolver(scope),
		resolver.NewCrossSymbolResolver(scalaLangName),
	)
	scala := resolver.NewScalaSymbolResolver(chain)
	return resolver.NewMemoSymbolResolver(scala)
}

// ResolveSymbol implements the resolver.SymbolResolver interface.
func (sl *scalaLang) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.Symbol, error) {
	return sl.knownImportResolver.ResolveSymbol(c, ix, from, lang, imp)
}
