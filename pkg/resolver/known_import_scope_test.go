package resolver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKnownImportScopeAdd(t *testing.T) {
	for name, tc := range map[string]struct {
		known KnownImport
		want  bool
	}{
		"degenerate": {},
		"not added": {
			known: KnownImport{Import: "nope"},
			want:  false,
		},
		"added": {
			known: KnownImport{Import: "a.b"},
			want:  true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := make(KnownImportScope)
			got := scope.Add(&tc.known)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestImportBasename(t *testing.T) {
	type result struct {
		Basename string
		Ok       bool
	}
	for name, tc := range map[string]struct {
		imp  string
		want result
	}{
		"degenerate": {},
		"typical": {
			imp:  "com.foo.Bar",
			want: result{"Bar", true},
		},
		"no dot": {
			imp: "com_foo_Bar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			basename, ok := importBasename(tc.imp)
			got := result{basename, ok}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
