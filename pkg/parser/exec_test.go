package parser

import (
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExecJS(t *testing.T) {
	for name, tc := range map[string]struct {
		dir          string
		args         []string
		in           io.Reader
		env          []string
		wantExitCode int
		wantStdout   string
		wantStderr   string
	}{
		"version": {
			dir:          ".",
			args:         []string{"--version"},
			wantStderr:   "",
			wantStdout:   "v19.1.0\n",
			wantExitCode: 0,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			gotExitCode, err := ExecJS(tc.dir, tc.args, tc.env, tc.in, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.wantExitCode, gotExitCode); diff != "" {
				t.Errorf("ExecJS exit code (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantStdout, stdout.String()); diff != "" {
				t.Errorf("ExecJS stdout (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantStderr, stderr.String()); diff != "" {
				t.Errorf("ExecJS stderr (-want +got):\n%s", diff)
			}
		})
	}
}
