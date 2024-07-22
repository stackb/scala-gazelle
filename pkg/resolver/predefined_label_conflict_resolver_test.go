package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
)

func TestPredefinedLabelConflictResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		symbol  resolver.Symbol
		rule    rule.Rule
		imports resolver.ImportMap
		imp     resolver.Import
		name    string
		from    label.Label
		want    *resolver.Symbol
		wantOk  bool
	}{
		"degenerate": {
			symbol: resolver.Symbol{
				Name:  "foo.Bar",
				Label: label.Label{Pkg: "foo", Name: "bar"},
			},
		},
		"takes conflict without a label": {
			symbol: resolver.Symbol{
				Name:  "foo.Bar",
				Label: label.Label{Pkg: "foo", Name: "bar"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "foo.Bar",
						Label: label.NoLabel,
					},
				},
			},
			want: &resolver.Symbol{
				Name:  "foo.Bar",
				Label: label.NoLabel,
			},
			wantOk: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := mocks.NewUniverse(t)
			resolver := resolver.PredefinedLabelConflictResolver{}
			got, gotOk := resolver.ResolveConflict(universe, &tc.rule, tc.imports, &tc.imp, &tc.symbol, tc.from)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ok (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
