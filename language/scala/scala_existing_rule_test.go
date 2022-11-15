package scala

import (
	"flag"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
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
				resolved.Set(l, imp, &ImportOrigin{Kind: ImportKindIndirect})
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
			sc := getOrCreateScalaConfig(c)
			sc.parseDirectives("", tc.directives)
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

			expr := makeLabeledListExpr(c, target.Kind(), shouldKeep, target.Attr("deps"), from, resolved)
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
		setup func(t *testing.T)
		want  bool
	}{
		"empty": {},
		"empty string": {
			deps: `    "",`,
			want: false,
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
			setup: func(t *testing.T) {
				fakeResolver := &fakeLabelOwnerResolver{}
				crossresolve.Resolvers().MustRegisterResolver("fake", fakeResolver)
			},
			deps: `    "@maven//:junit_junit",`,
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			getLabelOwners() // trigger lazy-build side-effect early
			if tc.setup != nil {
				tc.setup(t)
			}
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

			expr := listExpr.List[0]
			got := shouldKeep(expr)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("shouldKeep (-want +got):\n%s", diff)
			}
		})
	}
}

type fakeLabelOwnerResolver struct {
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (cr *fakeLabelOwnerResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (cr *fakeLabelOwnerResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// IsOwner implements the LabelOwner interface.
func (cr *fakeLabelOwnerResolver) IsOwner(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	return true
}

// CrossResolve implements the CrossResolver interface.
func (cr *fakeLabelOwnerResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	return nil
}
