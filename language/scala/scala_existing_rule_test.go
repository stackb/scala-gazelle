package scala

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	"github.com/stackb/scala-gazelle/pkg/crossresolve"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// This looks important: https://github.com/sbt/zinc/blob/7c796ce65217096ce71be986149b2e769f8b33af/internal/zinc-core/src/main/scala/sbt/internal/inc/Relations.scala

func TestResolveNameInFile(t *testing.T) {
	for name, tc := range map[string]struct {
		file index.ScalaFileSpec
		name string
		want string
	}{
		"degenerate": {
			want: ``,
		},
		"miss": {
			file: index.ScalaFileSpec{},
			name: "Bar",
			want: "",
		},
		"hit trait": {
			file: index.ScalaFileSpec{Traits: []string{"com.foo.Bar"}},
			name: "Bar",
			want: "com.foo.Bar",
		},
		"hit class": {
			file: index.ScalaFileSpec{Classes: []string{"com.foo.Bar"}},
			name: "Bar",
			want: "com.foo.Bar",
		},
		"hit object": {
			file: index.ScalaFileSpec{Objects: []string{"com.foo.Bar"}},
			name: "Bar",
			want: "com.foo.Bar",
		},
		"hit type": {
			file: index.ScalaFileSpec{Types: []string{"com.foo.Bar"}},
			name: "Bar",
			want: "com.foo.Bar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := resolveNameInFile(&tc.file)(tc.name)
			if tc.want == "" && !ok {
				return
			}
			if tc.want == "" && ok {
				t.Fatal("resolveNameInFile failed: expected miss")
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestResolveNameInFile (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveNameInLabelImportMap(t *testing.T) {
	for name, tc := range map[string]struct {
		resolved map[string]string
		name     string
		want     string
	}{
		"degenerate": {
			want: ``,
		},
		"miss": {
			name: "LazyLogging",
			want: "",
		},
		"hit": {
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			name: "LazyLogging",
			want: "com.typesafe.scalalogging.LazyLogging",
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved := make(LabelImportMap)
			for imp, origin := range tc.resolved {
				l, _ := label.Parse(origin)
				resolved.Set(l, imp, &ImportOrigin{Kind: ImportKindImplicit})
			}
			got, ok := resolveNameInLabelImportMap(resolved)(tc.name)
			if tc.want == "" && !ok {
				return
			}
			if tc.want == "" && ok {
				t.Fatal("resolvedInLabelImportMap failed: expected miss")
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestResolvedInLabelImportMap (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMakeLabeledListExpr(t *testing.T) {
	for name, tc := range map[string]struct {
		// prelude is an optional chunk of BUILD file content
		directives []rule.Directive
		// in is the existing rule
		in string
		// resolved is a mapping from import -> label
		resolved map[string]string
		// want is the expected rule appearance
		want string
	}{
		"degenerate": {
			in: `scala_library(name="test")`,
			want: `scala_library(
    name = "test",
    deps = [],
)
`,
		},
		"simple": {
			in: `scala_library(name="test")`,
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			want: `scala_library(
    name = "test",
    deps = ["@maven//:com_typesafe_scala_logging_scala_logging_2_12"],
)
`,
		},
		"simple+reason": {
			in:         `scala_library(name="test")`,
			directives: []rule.Directive{{Key: "scala_explain_dependencies", Value: "true"}},
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			want: `scala_library(
    name = "test",
    deps = [
        # com.typesafe.scalalogging.LazyLogging (comment)
        "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
    ],
)
`,
		},
		"simple+reason+deduplicate": {
			in:         `scala_library(name="test")`,
			directives: []rule.Directive{{Key: "scala_explain_dependencies", Value: "true"}},
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			want: `scala_library(
    name = "test",
    deps = [
        # com.typesafe.scalalogging.LazyLogging (comment)
        "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
    ],
)
`,
		},
		"keep": {
			in: `scala_library(
    name="test",
    deps=[
        ":foo",  # keep
    ],
)
`,
			directives: []rule.Directive{{Key: "scala_explain_dependencies", Value: "true"}},
			want: `scala_library(
    name = "test",
    deps = [
        ":foo",  # keep
    ],
)
`,
		},
		"keep+resolved": {
			in: `scala_library(
    name="test",
    deps=[
        ":foo",  # keep
    ],
)
`,
			directives: []rule.Directive{{Key: "scala_explain_dependencies", Value: "true"}},
			resolved: map[string]string{
				"com.typesafe.scalalogging.LazyLogging": "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
			},
			want: `scala_library(
    name = "test",
    deps = [
        ":foo",  # keep
        # com.typesafe.scalalogging.LazyLogging (comment)
        "@maven//:com_typesafe_scala_logging_scala_logging_2_12",
    ],
)
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config.New()
			index := &mockLabeledRuleIndex{}
			sc := getOrCreateScalaConfig(index, c, "")
			sc.parseDirectives(tc.directives)
			from := label.New("", "pkg", "rule")
			resolved := make(LabelImportMap)
			for imp, origin := range tc.resolved {
				l, _ := label.Parse(origin)
				resolved.Set(l, imp, &ImportOrigin{Kind: ImportKindComment})
			}

			file, err := rule.LoadData("<in-memory>", "BUILD", []byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if len(file.Rules) != 1 {
				t.Fatalf("expected single in rule, got %d", len(file.Rules))
			}
			target := file.Rules[0]

			keep := func(expr build.Expr) bool {
				return shouldKeep(expr, index.LookupRule)
			}
			expr := makeLabeledListExpr(c, target.Kind(), keep, target.Attr("deps"), from, resolved)
			target.SetAttr("deps", expr)
			want := tc.want
			got := printRule(target)
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

func TestShouldKeep(t *testing.T) {
	for name, tc := range map[string]struct {
		deps  string
		owner crossresolve.LabelOwner
		want  bool
	}{
		"empty string": {
			deps: `    "",`,
			want: true,
		},
		"keep empty string": {
			deps: `    "",  # keep`,
			want: true,
		},
		"unmanaged label": {
			deps: `    "@maven//:junit_junit",`,
			want: true,
		},
		"managed label": {
			owner: &repoLabelOwner{repo: "maven"},
			deps:  `    "@maven//:junit_junit_2_12",`,
			want:  false,
		},
		"managed scala_dep": {
			owner: &repoLabelOwner{repo: "maven"},
			deps:  `    scala_dep("@maven//:junit_junit"),`,
			want:  false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			index := &mockLabeledRuleIndex{}
			getLabelOwners() // trigger lazy-build side-effect early
			content := fmt.Sprintf(`
scala_library(
	name = "test",
	deps = [
		%s
	]
)`, tc.deps)
			file, err := rule.LoadData("<in-memory>", "BUILD", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			r := file.File.Rules("scala_library")[0]
			exprs := r.Attr("deps")
			listExpr := exprs.(*build.ListExpr)
			if len(listExpr.List) == 0 {
				return
			}

			if tc.owner == nil {
				tc.owner = &repoLabelOwner{}
			}
			expr := listExpr.List[0]
			got := shouldKeep(expr, index.LookupRule, tc.owner)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("shouldKeep (-want +got):\n%s", diff)
			}
		})
	}
}

type repoLabelOwner struct {
	repo string
}

// IsLabelOwner implements the LabelOwner interface.
func (cr *repoLabelOwner) IsLabelOwner(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	return from.Repo == cr.repo
}

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
			got := scalaDepLabel(expr)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("label (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommentUnresolvedImports(t *testing.T) {
	type testCase struct {
		// src is the rule source code
		src string
		// unresolved is the import map under test
		unresolved ImportOriginMap
		// want is the formatted output
		want string
	}
	for name, tc := range map[string]*testCase{
		"no srcs attribute": {
			unresolved: map[string]*ImportOrigin{
				"com.foo.Bar": NewDirectImportOrigin(&index.ScalaFileSpec{
					Filename: "Main.scala",
				}),
			},
			src: `
scala_library(
    name = "test",
    deps = [],
)`,
			want: `
scala_library(
    name = "test",
    deps = [],
)`,
		},
		"with srcs attribute": {
			unresolved: map[string]*ImportOrigin{
				"com.foo.Bar": NewDirectImportOrigin(&index.ScalaFileSpec{
					Filename: "Main.scala",
				}),
			},
			src: `
scala_library(
    name = "test",
    srcs = ["Main.scala"],
    deps = [],
)`,
			want: `
scala_library(
    name = "test",
    srcs =
    # unresolved: com.foo.Bar (direct from Main.scala)
    ["Main.scala"],
    deps = [],
)`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := mustLoadRule(t, tc.src)
			commentUnresolvedImports(tc.unresolved, r, "srcs")
			want := strings.TrimSpace(tc.want)
			got := ruleString(r)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("CommentUnresolvedImports() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustLoadRule(t *testing.T, content string) *rule.Rule {
	f, err := rule.LoadData("<in-memory>", "", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Rules) != 1 {
		t.Fatal("want 1 rule, got:", len(f.Rules))
	}
	return f.Rules[0]
}

func ruleString(r *rule.Rule) string {
	file := rule.EmptyFile("", "")
	r.Insert(file)
	return strings.TrimSpace(string(file.Format()))
}
