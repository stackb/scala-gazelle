package resolver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScopeMapAdd(t *testing.T) {
	for name, tc := range map[string]struct {
		symbol Symbol
		want   bool
	}{
		"degenerate": {},
		"not added": {
			symbol: Symbol{Name: "nope"},
			want:   false,
		},
		"added": {
			symbol: Symbol{Name: "a.b"},
			want:   true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := make(SymbolMap)
			got := scope.Add(&tc.symbol)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestSymbolBasename(t *testing.T) {
	type result struct {
		Basename string
		Ok       bool
	}
	for name, tc := range map[string]struct {
		name string
		want result
	}{
		"degenerate": {},
		"typical": {
			name: "com.foo.Bar",
			want: result{"Bar", true},
		},
		"no dot": {
			name: "com_foo_Bar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			basename, ok := symbolBasename(tc.name)
			got := result{basename, ok}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
