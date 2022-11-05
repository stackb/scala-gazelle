package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

func SkipTestGetScalaImportsFromRuleComment(t *testing.T) {
	for name, tc := range map[string]struct {
		source string
		want   []string
	}{
		"degenerate": {
			source: `scala_library(name="lib", deps=[])`,
			want:   nil,
		},
		"ok": {
			source: `
scala_library(
    name="lib",
    # scala-import: com.foo.Bar
    deps = [],
)
`,
			want: []string{"com.foo.Bar"},
		},
		"additional content ok": {
			source: `
scala_library(
    name="lib",
    # scala-import: com.foo.Bar // fixme
    deps = [],
)
`,
			want: []string{"com.foo.Bar"},
		},
		"not plural": {
			source: `
scala_library(
    name="lib",
    # scala-imports: com.foo.Bar
    deps = [],
)
`,
			want: nil,
		},
		"with comment": {
			source: `
scala_library(
    name="lib",
    # scala-import:com.foo.Bar   // needed by foo
    deps = [],
)
`,
			want: []string{"com.foo.Bar"},
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

			content := string(file.Format())
			t.Log("content:", content)

			if false {
				spew.Dump(file)
				if diff := cmp.Diff(tc.source, content); diff != "" {
					t.Errorf("content (-want +got):\n%s", diff)
				}
			}

			t.Logf("lhs: %+v", file.File.Stmt[0].(*build.CallExpr).List[1].(*build.AssignExpr).LHS)
			t.Logf("rhs: %+v", file.File.Stmt[0].(*build.CallExpr).List[1].(*build.AssignExpr).RHS)
			got := getScalaImportsFromRuleAttrComment("deps", "scala-import:", file.Rules[0])
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsSameImport(t *testing.T) {
	for name, tc := range map[string]struct {
		repoName string
		kind     string
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
			sc := newScalaConfig(c)
			got := isSameImport(sc, tc.kind, tc.from, tc.to)
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
		"does not strip do_not_import": {
			in:   []string{"@platform//:do_not_import"},
			want: []string{"@platform//:do_not_import"},
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
