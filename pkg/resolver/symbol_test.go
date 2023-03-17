package resolver

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
)

func TestSymbolConflicts(t *testing.T) {
	for name, tc := range map[string]struct {
		symbol    Symbol
		conflicts []*Symbol
		want      []*Symbol
	}{
		"degenerate": {},
		"typical case": {
			conflicts: []*Symbol{
				{
					Label: label.Label{Name: "a"},
				},
				{
					Label: label.Label{Name: "b"},
				},
			},
			want: []*Symbol{
				{
					Label: label.Label{Name: "a"},
				},
				{
					Label: label.Label{Name: "b"},
				},
			},
		},
		"ignores symbols without labels": {
			conflicts: []*Symbol{
				{
					Label: label.Label{Name: "a"},
				},
				{
					Label: label.NoLabel,
				},
			},
			want: []*Symbol{
				{
					Label: label.Label{Name: "a"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, sym := range tc.conflicts {
				tc.symbol.Conflict(sym)
			}
			got := tc.symbol.Conflicts
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
