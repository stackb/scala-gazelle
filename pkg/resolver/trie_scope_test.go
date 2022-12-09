package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func makeSymbol(typ sppb.ImportType, name string, from label.Label) *resolver.Symbol {
	return &resolver.Symbol{
		Type:     typ,
		Name:     name,
		Label:    label.NoLabel,
		Provider: "test",
	}
}

func TestTrieScope(t *testing.T) {

	for name, tc := range map[string]struct {
		symbols []*resolver.Symbol
		name    string
		want    *resolver.Symbol
	}{
		"degenerate": {},
		"miss": {
			name: "com.foo.Bar",
			want: nil,
		},
		"direct hit": {
			symbols: []*resolver.Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.foo.Bar",
			want: makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent class hit": {
			symbols: []*resolver.Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.foo.Bar.method",
			want: makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent package hit": {
			symbols: []*resolver.Symbol{
				makeSymbol(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.foo.Bar",
			want: makeSymbol(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent package miss": {
			symbols: []*resolver.Symbol{
				makeSymbol(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.bar.Baz",
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := resolver.NewTrieScope()
			for _, known := range tc.symbols {
				if err := scope.PutSymbol(known); err != nil {
					t.Fatal(err)
				}
			}
			got, _ := scope.GetSymbol(tc.name)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestTrieScopeGetSymbols(t *testing.T) {
	for name, tc := range map[string]struct {
		symbols []*resolver.Symbol
		prefix  string
		want    []*resolver.Symbol
	}{
		"degenerate": {},
		"miss": {
			prefix: "com.foo",
			want:   nil,
		},
		"completes known sorted": {
			symbols: []*resolver.Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "a.b.C", label.Label{Pkg: "a/b", Name: "scala_lib"}),
			},
			prefix: "com.foo",
			want: []*resolver.Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := resolver.NewTrieScope()
			for _, known := range tc.symbols {
				if err := scope.PutSymbol(known); err != nil {
					t.Fatal(err)
				}
			}
			got := scope.GetSymbols(tc.prefix)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
