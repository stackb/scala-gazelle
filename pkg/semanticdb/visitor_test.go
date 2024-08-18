package semanticdb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToImport(t *testing.T) {
	for name, tc := range map[string]struct {
		symbol string
		want   string
	}{
		"degenerate": {},
		"class": {
			symbol: "scala/Unit#",
			want:   "scala.Unit",
		},
		"package function": {
			symbol: "omnistac/euds/package.isFinraTraceEvent().(r)",
			want:   "omnistac.euds.package",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := toImport(tc.symbol)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
