package resolver_test

import (
	"sort"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func TestImportMapKeys(t *testing.T) {
	for name, tc := range map[string]struct {
		imports []*resolver.Import
		want    []string
	}{
		"degenerate": {
			want: []string{},
		},
		"maintains insert order properly": {
			imports: []*resolver.Import{
				{Imp: "c", Kind: parse.ImportKind_DIRECT},
				{Imp: "a", Kind: parse.ImportKind_EXTENDS},
				{Imp: "b", Kind: parse.ImportKind_IMPLICIT},
			},
			want: []string{"c", "a", "b"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			importMap := resolver.NewImportMap(tc.imports...)
			keys := importMap.Keys()
			if diff := cmp.Diff(tc.want, keys); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
			keys = []string{}
			for _, imp := range importMap.Values() {
				keys = append(keys, imp.Imp)
			}
			if diff := cmp.Diff(tc.want, keys); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}

		})
	}
}

func TestImportMapDeps(t *testing.T) {
	for name, tc := range map[string]struct {
		from    label.Label
		imports []*resolver.Import
		want    []label.Label
		wantErr error
	}{
		"degenerate": {
			want: []label.Label{},
		},
		"removes duplicates": {
			imports: []*resolver.Import{
				{
					Kind: parse.ImportKind_DIRECT,
					Imp:  "com.typesafe.scalalogging.LazyLogging",
					Symbol: &resolver.Symbol{
						Name:     "com.typesafe.scalalogging.LazyLogging",
						Provider: "maven",
						Label:    label.Label{Repo: "maven", Name: "com_typesafe_scalalogging"},
					},
				},
				{
					Kind: parse.ImportKind_DIRECT,
					Imp:  "com.typesafe.scalalogging.LazyLogging",
					Symbol: &resolver.Symbol{
						Name:     "com.typesafe.scalalogging.LazyLogging",
						Provider: "maven",
						Label:    label.Label{Repo: "maven", Name: "com_typesafe_scalalogging"},
					},
				},
			},
			from: label.Label{Pkg: "app", Name: "server"},
			want: []label.Label{
				{Repo: "maven", Name: "com_typesafe_scalalogging"},
			},
		},
		"removes self-imports": {
			imports: []*resolver.Import{
				{
					Kind: parse.ImportKind_DIRECT,
					Imp:  "com.typesafe.scalalogging.LazyLogging",
					Symbol: &resolver.Symbol{
						Name:     "com.typesafe.scalalogging.LazyLogging",
						Provider: "maven",
						Label:    label.Label{Repo: "maven", Name: "com_typesafe_scalalogging"},
					},
				},
				{
					Kind: parse.ImportKind_DIRECT,
					Imp:  "app.Server",
					Symbol: &resolver.Symbol{
						Name:     "app.Server",
						Provider: "source",
						Label:    label.Label{Relative: true, Name: "server"},
					},
				},
			},
			from: label.Label{Pkg: "app", Name: "server"},
			want: []label.Label{
				{Repo: "maven", Name: "com_typesafe_scalalogging"},
			},
		},
		"first-put-wins-semantics": {
			imports: []*resolver.Import{
				{
					Kind: parse.ImportKind_DIRECT,
					Imp:  "com.typesafe.scalalogging.LazyLogging",
					Symbol: &resolver.Symbol{
						Name:     "com.typesafe.scalalogging.LazyLogging",
						Provider: "maven",
						Label:    label.Label{Repo: "maven", Name: "com_typesafe_scalalogging"},
					},
				},
				{
					Kind: parse.ImportKind_DIRECT,
					Imp:  "com.typesafe.scalalogging.LazyLogging",
					Symbol: &resolver.Symbol{
						Name:     "com.typesafe.scalalogging.LazyLogging",
						Provider: "maven",
						Label:    label.Label{Repo: "maven", Name: "com_typesafe_scalalogging2"},
					},
				},
			},
			from: label.Label{Pkg: "app", Name: "server"},
			want: []label.Label{
				{Repo: "maven", Name: "com_typesafe_scalalogging"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			importMap := resolver.NewImportMap()
			for _, imp := range tc.imports {
				importMap.Put(imp)
			}
			labels := importMap.Deps(tc.from)
			got := make([]label.Label, len(labels))
			var i int
			for lbl := range labels {
				got[i] = lbl
				i++
			}
			sort.Slice(got, func(i, j int) bool {
				a := got[i]
				b := got[j]
				return a.String() < b.String()
			})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
