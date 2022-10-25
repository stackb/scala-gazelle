package scalaparse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseOptions(t *testing.T) {
	for name, tc := range map[string]struct {
		toolPath string
		args     []string
		want     *Options
	}{
		"degenerate": {
			want: &Options{
				RestoreEmbeddedFiles: true,
				NodeBinPath:          "faketmpdir/external/nodejs_darwin_amd64/bin/nodejs/bin/node",
				NodePath:             "faketmpdir/pkg/scalaparse",
				ScriptPath:           "faketmpdir/pkg/scalaparse/sourceindexer.js",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir := "faketmpdir"
			got, err := ParseOptions(tmpDir, tc.toolPath, tc.args)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseOptions (-want +got):\n%s", diff)
			}
		})
	}
}
