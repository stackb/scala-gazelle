package resolver

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestKnownImportRegistryTrie(t *testing.T) {
	for name, tc := range map[string]struct {
		known []*KnownImport
		imp   string
		want  *KnownImport
	}{
		"degenerate": {},
		"miss": {
			imp:  "com.foo.Bar",
			want: nil,
		},
		"direct hit": {
			known: []*KnownImport{
				{
					Type:   sppb.ImportType_CLASS,
					Import: "com.foo.Bar",
					Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
				},
			},
			imp: "com.foo.Bar",
			want: &KnownImport{
				Type:   sppb.ImportType_CLASS,
				Import: "com.foo.Bar",
				Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
			},
		},
		"parent class hit": {
			known: []*KnownImport{
				{
					Type:   sppb.ImportType_CLASS,
					Import: "com.foo.Bar",
					Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
				},
			},
			imp: "com.foo.Bar.method",
			want: &KnownImport{
				Type:   sppb.ImportType_CLASS,
				Import: "com.foo.Bar",
				Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
			},
		},
		"parent package hit": {
			known: []*KnownImport{
				{
					Type:   sppb.ImportType_PACKAGE,
					Import: "com.foo",
					Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
				},
			},
			imp: "com.foo.Bar",
			want: &KnownImport{
				Type:   sppb.ImportType_PACKAGE,
				Import: "com.foo",
				Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
			},
		},
		"parent package miss": {
			known: []*KnownImport{
				{
					Type:   sppb.ImportType_PACKAGE,
					Import: "com.foo",
					Label:  label.Label{Pkg: "com/foo", Name: "scala_lib"},
				},
			},
			imp:  "com.bar.Baz",
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			importRegistry := NewKnownImportRegistryTrie()
			for _, known := range tc.known {
				if err := importRegistry.PutKnownImport(known); err != nil {
					t.Fatal(err)
				}
			}
			got, _ := importRegistry.GetKnownImport(tc.imp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
