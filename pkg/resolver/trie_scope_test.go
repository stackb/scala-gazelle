package resolver

import (
	"fmt"
	"log"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func makeSymbol(typ sppb.ImportType, name string, from label.Label) *Symbol {
	return &Symbol{
		Type:     typ,
		Name:     name,
		Label:    label.NoLabel,
		Provider: "test",
	}
}

func TestTrieScope(t *testing.T) {

	for name, tc := range map[string]struct {
		symbols []*Symbol
		name    string
		want    *Symbol
	}{
		"degenerate": {},
		"miss": {
			name: "com.foo.Bar",
			want: nil,
		},
		"direct hit": {
			symbols: []*Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.foo.Bar",
			want: makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent class hit": {
			symbols: []*Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.foo.Bar.method",
			want: makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent package hit": {
			symbols: []*Symbol{
				makeSymbol(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.foo.Bar",
			want: makeSymbol(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
		},
		"parent package miss": {
			symbols: []*Symbol{
				makeSymbol(sppb.ImportType_PACKAGE, "com.foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			name: "com.bar.Baz",
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := NewTrieScope()
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
		symbols []*Symbol
		prefix  string
		want    []*Symbol
	}{
		"degenerate": {},
		"miss": {
			prefix: "com.foo",
			want:   nil,
		},
		"completes known sorted": {
			symbols: []*Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "a.b.C", label.Label{Pkg: "a/b", Name: "scala_lib"}),
			},
			prefix: "com.foo",
			want: []*Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := NewTrieScope()
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

func TestTrieScopeGetScope(t *testing.T) {
	for name, tc := range map[string]struct {
		symbols []*Symbol
		prefix  string
		names   []string
		want    []*Symbol
	}{
		"degenerate": {},
		"completes known sorted": {
			symbols: []*Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
			prefix: "com.foo",
			names:  []string{"Foo", "Bar", "Nope"},
			want: []*Symbol{
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Foo", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
				makeSymbol(sppb.ImportType_CLASS, "com.foo.Bar", label.Label{Pkg: "com/foo", Name: "scala_lib"}),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var scope Scope
			scope = NewTrieScope()
			for _, known := range tc.symbols {
				if err := scope.PutSymbol(known); err != nil {
					t.Fatal(err)
				}
			}
			scope, _ = scope.GetScope(tc.prefix)
			var got []*Symbol
			for _, name := range tc.names {
				if symbol, ok := scope.GetSymbol(name); ok {
					got = append(got, symbol)
				} else {
					t.Log("scope not found:", name)
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

type result struct {
	Segment string
	Path    int
}

func TestImportSegmenter(t *testing.T) {
	for name, tc := range map[string]struct {
		want []result
	}{
		"degenerate": {
			want: []result{
				{Segment: "degenerate", Path: -1},
			},
		},
		"a": {
			want: []result{
				{Segment: "a", Path: -1},
			},
		},
		"aaa": {
			want: []result{
				{Segment: "aaa", Path: -1},
			},
		},
		"a.b.c": {
			want: []result{
				{Segment: "a", Path: 2},
				{Segment: "b", Path: 4},
				{Segment: "c", Path: -1},
			},
		},
		"a..b": {
			want: []result{
				{Segment: "a", Path: 2},
				{Segment: ".b", Path: -1},
			},
		},
		"a.": {
			want: []result{
				{Segment: "a", Path: 2},
			},
		},
		"a..": {
			want: []result{
				{Segment: "a", Path: 2},
				{Segment: ".", Path: -1},
			},
		},
		"a...b": {
			want: []result{
				{Segment: "a", Path: 2},
				{Segment: ".", Path: 4},
				{Segment: "b", Path: -1},
			},
		},
		"✅.❌": {
			want: []result{
				{Segment: "✅", Path: 4},
				{Segment: "❌", Path: -1},
			},
		},
		"✅✅✅.❌❌❌": {
			want: []result{
				{Segment: "✅✅✅", Path: 10},
				{Segment: "❌❌❌", Path: -1},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var got []result
			for part, i := importSegmenter(name, 0); part != ""; part, i = importSegmenter(name, i) {
				got = append(got, result{part, i})
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func ExampleTrieScope_String_empty() {
	scope := NewTrieScope()
	fmt.Println(scope)
	// output:
	//
}

func ExampleTrieScope_String() {
	scope := NewTrieScope()

	for _, symbol := range []*Symbol{
		{
			Type:     sppb.ImportType_PACKAGE,
			Name:     "java.lang",
			Provider: "java",
		},
		{
			Type:     sppb.ImportType_CLASS,
			Name:     "java.lang.String",
			Provider: "java",
		},
		{
			Type:     sppb.ImportType_PACKAGE,
			Name:     "scala",
			Provider: "java",
		},
		{
			Type:     sppb.ImportType_CLASS,
			Name:     "scala.Any",
			Provider: "java",
		},
	} {
		if err := scope.PutSymbol(symbol); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println(scope)
	// output:
	//
	// java <nil>
	// └ lang (java.lang<PACKAGE> //:<java>)
	//   └ String (java.lang.String<CLASS> //:<java>)
	// scala (scala<PACKAGE> //:<java>)
	// └ Any (scala.Any<CLASS> //:<java>)
}
