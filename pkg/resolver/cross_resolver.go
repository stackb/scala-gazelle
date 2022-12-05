package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// CrossResolver implements Resolver using the resolve.RuleIndex (which uses the
// gazelle cross resolver infrastructure).
type CrossResolver struct {
	lang string
}

func NewCrossResolver(lang string) *CrossResolver {
	return &CrossResolver{lang}
}

// ResolveKnownImport implements the KnownImportResolver interface
func (sr *CrossResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	matches := ix.FindRulesByImportWithConfig(c, resolve.ImportSpec{Lang: lang, Imp: imp}, sr.lang)
	switch len(matches) {
	case 0:
		return nil, ErrImportNotFound
	case 1:
		return &KnownImport{
			Type:   sppb.ImportType_CROSS_RESOLVE,
			Import: imp,
			Label:  matches[0].Label,
		}, nil
	default:
		return nil, NewImportAmbiguousError(imp, matches)
	}
}
