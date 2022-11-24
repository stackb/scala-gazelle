package scala

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestImportOriginString(t *testing.T) {
	for name, tc := range map[string]struct {
		origin ImportOrigin
		want   string
	}{
		"degenerate": {
			want: "foo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := tc.origin.String()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
