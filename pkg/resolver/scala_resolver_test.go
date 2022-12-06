package resolver

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestScalaResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		lang string
		from label.Label
		imp  string
		want string
	}{
		"degenerate": {},
		"unchanged": {
			lang: "scala",
			from: label.Label{Pkg: "src", Name: "scala"},
			known: []*KnownImport{
				{
					Type:   sppb.ImportType_CLASS,
					Import: "com.foo.Bar",
					Label:  label.Label{Pkg: "lib", Name: "scala_lib"},
				},
			},
			imp: &Import{
				Kind: sppb.ImportKind_DIRECT,
				Imp:  "com.foo.Bar",
			},
			want: "com.foo.bar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			importRegistry := NewKnownImportRegistryTrie()
			for _, known := range tc.known {
				if err := importRegistry.PutKnownImport(known); err != nil {
					t.Fatal(err)
				}
			}

			next := &mockResolver{}

			rslv := NewScalaResolver(next)
			c := config.New()

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)

			rslv.ResolveKnownImport(c, ix, tc.from, tc.lang, tc.imp)

			got := next.gotImp

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

type mockResolver struct {
	wantImport *KnownImport
	wantErr    error
	gotImp     string
}

func (r *mockResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	r.gotImp = imp
	return r.wantImport, r.wantErr
}
