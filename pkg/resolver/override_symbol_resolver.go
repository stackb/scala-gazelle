package resolver

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

const overrideProviderName = "override"

// OverrideSymbolResolver implements Resolver for gazelle:resolve directives.
type OverrideSymbolResolver struct {
	lang string
}

func NewOverrideSymbolResolver(lang string) *OverrideSymbolResolver {
	return &OverrideSymbolResolver{lang}
}

// ResolveSymbol implements the SymbolResolver interface
func (sr *OverrideSymbolResolver) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*Symbol, bool) {
	if strings.HasPrefix(imp, "org.json4s") {
		log.Println(from, "OverrideSymbolResolver.ResolveSymbol", imp)
	}
	if imp == "org.json4s.JValue" {
		log.Println(from, "OverrideSymbolResolver.ResolveSymbol", imp)
	}
	if to, ok := resolve.FindRuleWithOverride(c, resolve.ImportSpec{Lang: lang, Imp: imp}, sr.lang); ok {
		return NewSymbol(sppb.ImportType_OVERRIDE, imp, overrideProviderName, to), true
	}
	return nil, false
}
