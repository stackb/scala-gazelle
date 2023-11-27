package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// newUniverseResolver constructs the top-level symbol resolver.
func newUniverseResolver(scope resolver.Scope, packages resolver.Scope) resolver.SymbolResolver {
	chain := resolver.NewChainSymbolResolver(
		// override resolver is the least performant!
		resolver.NewMemoSymbolResolver(resolver.NewOverrideSymbolResolver(scalaLangName)),
		resolver.NewScopeSymbolResolver(scope),
		resolver.NewCrossSymbolResolver(scalaLangName),
		resolver.NewScalaSymbolResolver(resolver.NewScopeSymbolResolver(packages)),
	)
	scala := resolver.NewScalaSymbolResolver(chain)
	return resolver.NewMemoSymbolResolver(scala)
}

// ResolveSymbol implements the resolver.SymbolResolver interface.
func (sl *scalaLang) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.Symbol, bool) {
	// if strings.HasPrefix(imp, "org.json4s") {
	// 	log.Println(from, "scalaLang.ResolveSymbol", imp)
	// }
	return sl.symbolResolver.ResolveSymbol(c, ix, from, lang, imp)
}
