package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestReadBuildEvents tests the .
func TestReadBuildEventsIn(t *testing.T) {
	for name, tc := range map[string]struct {
		content string
		want    []*BuildEvent
		wantErr error
	}{
		"degenerate case": {},
		"progress": {
			content: `
{"id":{"progress":{"opaqueCount":145}},"progress":{"stderr":"foo"}}
`,
			want: []*BuildEvent{
				{
					Progress: &Progress{Stderr: `foo`},
				},
			},
		},
		"action": {
			content: `
{
	"id": {
		"actionCompleted": {
			"primaryOutput": "bazel-out/stable-status.txt",
			"configuration": {
				"id": "system"
			}
		}
	},
	"action": {
		"label": "//com/foo:bar",
		"stderr": {
			"URI": "file:///path/to/stderr-1"
		}
	}
}
`,
			want: []*BuildEvent{
				{
					Action: &Action{
						Label: "//com/foo:bar",
						Stderr: File{
							URI: "file:///path/to/stderr-1",
						},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := readBuildEventsIn(strings.NewReader(tc.content))

			if diff := cmp.Diff(tc.wantErr, err); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
