package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// OverrideResolver implements Resolver for gazelle:resolve directives.
type OverrideResolver struct {
	lang string
}

func NewOverrideResolver(lang string) *OverrideResolver {
	return &OverrideResolver{lang}
}

// ResolveKnownImport implements the KnownImportResolver interface
func (sr *OverrideResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	if to, ok := resolve.FindRuleWithOverride(c, resolve.ImportSpec{Lang: lang, Imp: imp}, sr.lang); ok {
		return &KnownImport{
			Type:   sppb.ImportType_OVERRIDE,
			Import: imp,
			Label:  to,
		}, nil
	}
	return nil, ErrImportNotFound
}
