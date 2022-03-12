package scala

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func TestMakeLabeledListExpr(t *testing.T) {
	for name, tc := range map[string]struct {
		// prelude is an optional chunk of BUILD file content
		directives []rule.Directive
		// resolved is a mapping from import -> label
		resolved map[string]string
		// want is the expected rule appearance
		want string
	}{
		"degenerate": {
			want: `testkind(
    name = "testname",
    deps = [],
)
`,
		},
		"simple": {
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			want: `testkind(
    name = "testname",
    deps = ["@maven//:com_typesafe_scala_logging_scala_logging_2_12"],
)
`,
		},
		"simple+reason": {
			directives: []rule.Directive{{"scala_explain_dependencies", "true"}},
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			want: `testkind(
    name = "testname",
    deps = [
        # com.typesafe.scalalogging.LazyLogging
        "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
    ],
)
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config.New()
			sc := getOrCreateScalaConfig(c)
			sc.parseDirectives("", tc.directives)
			from := label.New("", "pkg", "rule")
			resolved := make(labelImportMap)
			for imp, origin := range tc.resolved {
				l, _ := label.Parse(origin)
				resolved.Set(l, imp)
			}
			expr := makeLabeledListExpr(c, from, resolved)
			r := rule.NewRule("testkind", "testname")
			r.SetAttr("deps", expr)
			want := tc.want
			got := printRule(r)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("TestMakeLabeledListExpr (-want +got):\n%s", diff)
			}
		})
	}
}

func printRule(rules ...*rule.Rule) string {
	file := rule.EmptyFile("", "")
	for _, r := range rules {
		r.Insert(file)
	}
	return string(file.Format())
}
