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
		"skip non-named labels": {
			repoName: "foo",
			rel:      "a",
			from:     label.New("bar", "", ""),
			want:     label.NoLabel,
		},
		"makes package absolute": {
			rel:  "a",
			from: label.New("", "", "test"),
			want: label.New("", "a", "test"),
		},
		"makes repo absolute": {
			repoName: "foo",
			from:     label.New("", "", "test"),
			want:     label.New("foo", "", "test"),
		},
		"absolute remains unchanged": {
			from: label.New("bar", "b", "test"),
			want: label.New("bar", "b", "test"),
		},
		"absolute without package unchanged": {
			repoName: "foo",
			rel:      "a",
			from:     label.New("bar", "", "test"),
			want:     label.New("bar", "", "test"),
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
