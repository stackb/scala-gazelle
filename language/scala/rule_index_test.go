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
		imp      resolve.ImportSpec
		provided map[resolve.ImportSpec]*crossresolve.ImportProvider
		want     label.Label
	}{
		// "degenerate": {},
		// "fully-qualified": {
		// 	imp: resolve.ImportSpec{Lang: "java", Imp: "scala.util.Success"},
		// 	provided: map[resolve.ImportSpec]*crossresolve.ImportProvider{
		// 		{Lang: "java", Imp: "scala.util.Success"}: {
		// 			Type:  "maven",
		// 			Label: label.Label{Repo: "maven", Name: "org_scala_lang_scala_library"},
		// 		},
		// 	},
		// 	want: label.Label{Repo: "maven", Name: "org_scala_lang_scala_library"},
		// },
		"package-qualified": {
			imp: resolve.ImportSpec{Lang: "java", Imp: "scala.util"},
			provided: map[resolve.ImportSpec]*crossresolve.ImportProvider{
				{Lang: "java", Imp: "scala.util.Success"}: {
					Type:  "maven",
					Label: label.Label{Repo: "maven", Name: "org_scala_lang_scala_library"},
				},
			},
			want: label.Label{Repo: "maven", Name: "org_scala_lang_scala_library"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			lang := NewLanguage().(*scalaLang)
			for imp, provider := range tc.provided {
				lang.recordImport(imp, provider)
			}
			provider, _ := lang.LookupImport(tc.imp)
			var got label.Label
			if provider != nil {
				got = provider.Label
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
