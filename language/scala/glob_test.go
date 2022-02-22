package scala

import (
	"fmt"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"
)

// TestParseGlob tests the parsing of a starlark glob.
func TestParseGlob(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		want rule.GlobValue
	}{
		"empty glob": {
			text: `glob()`,
			want: rule.GlobValue{},
		},
		"default include list - empty": {
			text: `glob([])`,
			want: rule.GlobValue{},
		},
		"default include list - one pattern": {
			text: `glob(["a.scala"])`,
			want: rule.GlobValue{Patterns: []string{"a.scala"}},
		},
		"default include list - two patterns": {
			text: `glob(["a.scala", "b.scala"])`,
			want: rule.GlobValue{Patterns: []string{"a.scala", "b.scala"}},
		},
		"exclude list - single exclude": {
			text: `glob([], exclude=["c.scala"])`,
			want: rule.GlobValue{Excludes: []string{"c.scala"}},
		},
		"complex value - not supported": {
			text: `glob(get_include_list(), get_exclude_list())`,
			want: rule.GlobValue{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			content := fmt.Sprintf("test_rule(srcs = %s)", tc.text)
			file, err := build.Parse("BUILD", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			r := file.Rules("test_rule")[0]

			got := parseGlob(r.Attr("srcs").(*build.CallExpr))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseGlob (-want +got):\n%s", diff)
			}
		})
	}
}
