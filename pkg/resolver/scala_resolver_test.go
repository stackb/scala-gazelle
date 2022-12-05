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
		lang    string
		from    label.Label
		known   []*KnownImport
		imp     *Import
		want    *Import
		wantErr error
	}{
		"degenerate": {},
		"resolve success": {
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
			want: &Import{
				Kind: sppb.ImportKind_DIRECT,
				Imp:  "com.foo.Bar",
				Known: &KnownImport{
					Type:   sppb.ImportType_CLASS,
					Import: "com.foo.Bar",
					Label:  label.Label{Pkg: "lib", Name: "scala_lib"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			importRegistry := NewKnownImportRegistryTrie()
			for _, known := range tc.known {
				if err := importRegistry.PutKnownImport(known); err != nil {
					t.Fatal(err)
				}
			}

			rslv := NewScalaResolver("scala", importRegistry)
			c := config.New()

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)

			rslv.ResolveImport(c, ix, tc.from, tc.lang, tc.imp)

			if diff := cmp.Diff(tc.want, tc.imp); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
