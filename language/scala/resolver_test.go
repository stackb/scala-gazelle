package scala

import (
	"testing"

	"github.com/google/go-cmp/cmp"

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
