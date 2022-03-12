package scala

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func TestGetScalaImportsFromRuleComment(t *testing.T) {
	for name, tc := range map[string]struct {
		source string
		want   []string
	}{
		"degenerate": {
			source: `scala_library(name="lib")`,
			want:   nil,
		},
		"ok": {
			source: `
# scala-import: com.foo.Bar
scala_library(
	name="lib",
)
`,
			want: []string{"com.foo.Bar"},
		},
		"additional content ok": {
			source: `
# scala-import: com.foo.Bar // fixme
scala_library(
	name="lib",
)
`,
			want: []string{"com.foo.Bar"},
		},
		"not plural": {
			source: `
# scala-imports: com.foo.Bar
scala_library(
	name="lib",
)
`,
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			file, err := rule.LoadData("", "", []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(file.Rules) != 1 {
				t.Fatal("test case should decare one rule")
			}
			got := getScalaImportsFromRuleComment(file.Rules[0])
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsSameImport(t *testing.T) {
	for name, tc := range map[string]struct {
		repoName string
		from, to label.Label
		want     bool
	}{
		"degenerate": {
			want: true,
		},
		"equal": {
			from: label.New("corp", "pkg", "name"),
			to:   label.New("corp", "pkg", "name"),
			want: true,
		},
		"different": {
			repoName: "corp",
			from:     label.New("other", "pkg", "name"),
			to:       label.New("other", "pkg", "name"),
			want:     true,
		},
		"both internal labels": {
			repoName: "corp",
			from:     label.New("", "pkg", "name"),
			to:       label.New("", "pkg", "name"),
			want:     true,
		},
		"from has repo": {
			repoName: "corp",
			from:     label.New("corp", "pkg", "name"),
			to:       label.New("", "pkg", "name"),
			want:     true,
		},
		"to has repo": {
			repoName: "corp",
			from:     label.New("", "pkg", "name"),
			to:       label.New("corp", "pkg", "name"),
			want:     true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config.New()
			c.RepoName = tc.repoName
			got := isSameImport(c, tc.from, tc.to)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestDedupLabels(t *testing.T) {
	for name, tc := range map[string]struct {
		in   []string
		want []string
	}{
		"degenerate": {
			want: []string{},
		},
		"deduplicates": {
			in:   []string{"//a", "//b", "//a"},
			want: []string{"//a", "//b"},
		},
		"preserves ordering": {
			in:   []string{"//b", "//a", "//a", "//c"},
			want: []string{"//b", "//a", "//c"},
		},
		"strips platform": {
			in:   []string{"@platform//:do_not_import"},
			want: []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			input := make([]label.Label, len(tc.in))
			for i, l := range tc.in {
				lbl, err := label.Parse(l)
				if err != nil {
					t.Fatal(err)
				}
				input[i] = lbl
			}
			output := dedupLabels(input)
			got := make([]string, len(output))
			for i, o := range output {
				got[i] = o.String()
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
