package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// CrossSymbolResolver implements Resolver using the resolve.RuleIndex (which uses the
// gazelle cross resolver infrastructure).
type CrossSymbolResolver struct {
	lang string
}

func NewCrossSymbolResolver(lang string) *CrossSymbolResolver {
	return &CrossSymbolResolver{lang}
}

// ResolveSymbol implements the SymbolResolver interface
func (sr *CrossSymbolResolver) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*Symbol, bool) {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: lang, Imp: imp}, sr.lang)
	switch len(matches) {
	case 0:
		return nil, false
	case 1:
		return NewSymbol(sppb.ImportType_CROSS_RESOLVE, imp, "cross-resolve", matches[0].Label), true
	default:
		sym := NewSymbol(sppb.ImportType_CROSS_RESOLVE, imp, "cross-resolve", matches[0].Label)
		for _, match := range matches[1:] {
			conflict := NewSymbol(sppb.ImportType_CROSS_RESOLVE, imp, "cross-resolve", match.Label)
			sym.Conflicts = append(sym.Conflicts, conflict)
		}
		return sym, true
	}
}
