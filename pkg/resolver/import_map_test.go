package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

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
			for i, v := range labels {
				got[i] = v.Label
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
