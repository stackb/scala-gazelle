package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProgressParseStderr(t *testing.T) {
	for name, tc := range map[string]struct {
		// prelude is an optional chunk of BUILD file content
		stderr string
		// wantErr is an error to expect
		wantErr error
		// want is the expected migration IDs
		want []string
	}{
		"degenerate case": {
			want: []string{},
		},
		"no such target": {
			stderr: `
ERROR: /Users/i868039/go/src/github.com/Omnistac/unity/omnistac/postswarm/BUILD.bazel:177:18: no such target '//trumid/ats/common/proto:scala_proto': target 'scala_proto' not declared in package 'trumid/ats/common/proto' (did you mean 'java_proto'?) defined by /Users/i868039/go/src/github.com/Omnistac/unity/trumid/ats/common/proto/BUILD.bazel and referenced by '//omnistac/postswarm:flaky.it'
`,
			want: []string{"nst:filename=/Users/i868039/go/src/github.com/Omnistac/unity/omnistac/postswarm/BUILD.bazel,dep://trumid/ats/common/proto:scala_proto,from://omnistac/postswarm:flaky.it"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &config{}
			env := newEnv(cfg)
			p := &Progress{tc.stderr}

			mm, err := p.parseStderr(env)

			if diff := cmp.Diff(tc.wantErr, err); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}

			got := make([]string, len(mm))
			for i, m := range mm {
				got[i] = m.ID()
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
