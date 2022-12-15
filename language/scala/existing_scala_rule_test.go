package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
)

func TestScalaDepLabel(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want label.Label
	}{
		"degenerate": {
			in: `
test(
	expr = "",
)
			`,
			want: label.NoLabel,
		},
		"invalid label": {
			in: `
test(
	expr = "@@@",
)
			`,
			want: label.NoLabel,
		},
		"valid label": {
			in: `
test(
	expr = "@foo//bar:baz",
)
			`,
			want: label.New("foo", "bar", "baz"),
		},
		"invalid callexpr": {
			in: `
test(
	expr = fn("@foo//bar:baz"),
)
			`,
			want: label.NoLabel,
		},
		"valid callexpr": {
			in: `
test(
	expr = scala_dep("@foo//bar:baz"),
)
			`,
			want: label.New("foo", "bar", "baz"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			file, err := rule.LoadData("<in-memory>", "BUILD", []byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if len(file.Rules) != 1 {
				t.Fatalf("expected single in rule, got %d", len(file.Rules))
			}
			target := file.Rules[0]
			expr := target.Attr("expr")
			got := labelFromDepExpr(expr)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("label (-want +got):\n%s", diff)
			}
		})
	}
}
