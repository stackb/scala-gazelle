package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

type mockRuleIndex struct {
	// from records the argument given to LookupRule
	from label.Label
}

// LookupRule implements part of the crossresolve.RuleIndex interface
func (m *mockRuleIndex) LookupRule(from label.Label) (*rule.Rule, bool) {
	m.from = from
	return nil, false
}

// LookupImport imp resolve.ImportSpec) (*crossresolve.ImportProvider, bool) {
func (m *mockRuleIndex) LookupImport(imp resolve.ImportSpec) (*crossresolve.ImportProvider, bool) {
	return nil, false
}

func TestLookupImport(t *testing.T) {
	for name, tc := range map[string]struct {
		imp  resolve.ImportSpec
		want *crossresolve.ImportProvider
	}{
		"degenerate": {},
	} {
		t.Run(name, func(t *testing.T) {
			lang := NewLanguage().(*scalaLang)
			got, _ := lang.LookupImport(tc.imp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
