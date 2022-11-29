package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
)

func TestScalaConfigLookupRule(t *testing.T) {
	for name, tc := range map[string]struct {
		repoName string
		rel      string
		from     label.Label
		want     label.Label
	}{
		"degenerate": {
			from: label.NoLabel,
			want: label.NoLabel,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config.New()
			c.RepoName = tc.repoName
			index := &mockRuleIndex{}
			sc := newScalaConfig(index, c, tc.rel)
			sc.LookupRule(tc.from)
			got := index.from
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
