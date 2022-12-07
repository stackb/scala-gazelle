package scala

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

type fakeImportResolver struct {
	getKnownRuleFromArgument label.Label
}

func (r *fakeImportResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.KnownImport, error) {
	return nil, fmt.Errorf("unimplemented")
}

// KnownImportProviders implements part of the
// resolver.KnownImportProviderRegistry interface.
func (r *fakeImportResolver) KnownImportProviders() []resolver.KnownImportProvider {
	return nil
}

// AddKnownImportProvider implements part of the
// resolver.KnownImportProviderRegistry interface.
func (r *fakeImportResolver) AddKnownImportProvider(provider resolver.KnownImportProvider) error {
	return nil
}

// GetKnownImport implements part of the resolver.KnownImportRegistry interface.
func (r *fakeImportResolver) GetKnownImport(imp string) (*resolver.KnownImport, bool) {
	return nil, false
}

// PutKnownImport implements part of the resolver.KnownImportRegistry interface.
func (r *fakeImportResolver) PutKnownImport(known *resolver.KnownImport) error {
	return nil
}

// GetKnownRule implements part of the resolver.KnownRuleRegistry interface.
func (r *fakeImportResolver) GetKnownRule(from label.Label) (*rule.Rule, bool) {
	r.getKnownRuleFromArgument = from
	return nil, false
}

// PutKnownRule implements part of the resolver.KnownRuleRegistry interface.
func (r *fakeImportResolver) PutKnownRule(from label.Label, rule *rule.Rule) error {
	return nil
}
