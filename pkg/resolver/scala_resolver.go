package resolver

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// ScalaResolver implements Resolver for scala imports.
type ScalaResolver struct {
	// the language that should be used when resolving via the resolve.RuleIndex.
	lang string
	// known is the known import registry
	known KnownImportRegistry
}

func NewScalaResolver(lang string, known KnownImportRegistry) *ScalaResolver {
	return &ScalaResolver{lang: lang, known: known}
}

// ResolveImports implements the ImportResolver interface
func (sr *ScalaResolver) ResolveImports(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imports ...*Import) {
	for _, imp := range imports {
		sr.resolveImport(c, ix, from, lang, imp, imp.Imp)
	}
}

func (sr *ScalaResolver) resolveImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp *Import, symbol string) {

	if ok := sr.resolveOverride(c, ix, from, lang, imp, symbol); ok {
		return
	}

	if ok := sr.resolveKnown(c, ix, from, lang, imp, symbol); ok {
		return
	}

	if ok := sr.resolveWithIndex(c, ix, from, lang, imp, symbol); ok {
		// TODO: possibly disambuguate error here
		return
	}

	// if this is a _root_ import, try without
	if strings.HasPrefix(symbol, "_root_.") {
		sr.resolveImport(c, ix, from, lang, imp, strings.TrimPrefix(symbol, "_root_."))
		return
	}

	// if this is a _root_ import, try without
	if strings.HasPrefix(symbol, "._") {
		sr.resolveImport(c, ix, from, lang, imp, strings.TrimSuffix(symbol, "._"))
		return
	}

	// no luck
	imp.Error = ErrImportNotFound
}

func (sr *ScalaResolver) resolveOverride(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp *Import, symbol string) bool {
	if to, ok := resolve.FindRuleWithOverride(c, resolve.ImportSpec{Lang: lang, Imp: symbol}, lang); ok {
		imp.Known = &KnownImport{
			Type:   sppb.ImportType_OVERRIDE,
			Import: symbol,
			Label:  to,
		}
		return true
	}
	return false
}

func (sr *ScalaResolver) resolveKnown(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp *Import, symbol string) bool {
	if known, ok := sr.known.GetKnownImport(symbol); ok {
		imp.Known = known
		return true
	}
	return false
}

func (sr *ScalaResolver) resolveWithIndex(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp *Import, symbol string) bool {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: lang, Imp: symbol}, sr.lang)
	switch len(matches) {
	case 0:
		return false
	case 1:
		imp.Known = &KnownImport{
			Type:   sppb.ImportType_CROSS_RESOLVE,
			Import: symbol,
			Label:  matches[0].Label,
		}
		return true
	default:
		imp.Error = NewImportAmbiguousError(symbol, matches)
		return true
	}
}

// DeduplicateLabels deduplicates labels but keeps existing ordering.
func DeduplicateLabels(in []label.Label) (out []label.Label) {
	seen := make(map[label.Label]bool)
	for _, l := range in {
		if seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}
