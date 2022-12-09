package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func makeKnownImport(typ sppb.ImportType, imp string, from label.Label) *resolver.KnownImport {
	return &resolver.KnownImport{
		Type:     typ,
		Import:   imp,
		Label:    label.NoLabel,
		Provider: "test",
	}
}

func TestKnownImportRegistryTrie(t *testing.T) {

	for name, tc := range map[string]struct {
		known []*resolver.KnownImport
		imp   string
		want  *resolver.KnownImport
	}{
		"degenerate": {},
		"miss": {
			imp:  "com.foo.Bar",
			want: nil,
		},
		"direct hit": {
			known: []*resolver.KnownImport{
				makeKnownImport(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			imp:  "com.foo.Bar",
			want: makeKnownImport(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent class hit": {
			known: []*resolver.KnownImport{
				makeKnownImport(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			imp:  "com.foo.Bar.method",
			want: makeKnownImport(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent package hit": {
			known: []*resolver.KnownImport{
				makeKnownImport(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			imp:  "com.foo.Bar",
			want: makeKnownImport(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent package miss": {
			known: []*resolver.KnownImport{
				makeKnownImport(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			imp:  "com.bar.Baz",
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			importRegistry := resolver.NewKnownImportRegistryTrie()
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

func TestKnownImportRegistryTrieGetKnownImports(t *testing.T) {
	for name, tc := range map[string]struct {
		known  []*resolver.KnownImport
		prefix string
		want   []*resolver.KnownImport
	}{
		"degenerate": {},
		"miss": {
			prefix: "com.foo",
			want:   nil,
		},
		"completes known sorted": {
			known: []*resolver.KnownImport{
				makeKnownImport(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeKnownImport(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeKnownImport(sppb.ImportType_CLASS, "a.b.C", label.Label{Pkg: "a/b", Name: "scala_lib"}),
			},
			prefix: "com.foo",
			want: []*resolver.KnownImport{
				makeKnownImport(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeKnownImport(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			importRegistry := resolver.NewKnownImportRegistryTrie()
			for _, known := range tc.known {
				if err := importRegistry.PutKnownImport(known); err != nil {
					t.Fatal(err)
				}
			}
			got := importRegistry.GetKnownImports(tc.prefix)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
