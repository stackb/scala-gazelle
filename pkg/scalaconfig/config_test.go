package scalaconfig

import (
	"fmt"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func NewTestScalaConfig(t *testing.T, universe resolver.Universe, rel string, dd ...rule.Directive) (*Config, error) {
	c := config.New()
	sc := New(zerolog.New(os.Stderr), universe, c, rel)
	err := sc.ParseDirectives(dd)
	return sc, err
}

func TestScalaConfigParseDirectives(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       *Config
	}{
		"degenerate": {
			want: &Config{
				rules:             map[string]*scalarule.Config{},
				annotations:       map[debugAnnotation]any{},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
			},
		},
		"annotation after rule": {
			directives: []rule.Directive{
				{Key: "scala_rule", Value: "scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary"},
				{Key: "scala_debug", Value: "imports"},
			},
			want: &Config{
				rules: map[string]*scalarule.Config{
					"scala_binary": {
						Deps:           map[string]bool{},
						Attrs:          map[string]map[string]bool{},
						Options:        map[string]bool{},
						Enabled:        true,
						Implementation: "@io_bazel_rules_scala//scala:scala.bzl%scala_binary",
						Name:           "scala_binary",
					},
				},
				annotations: map[debugAnnotation]any{
					DebugImports: nil,
				},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
			},
		},
		"rule after annotation": {
			directives: []rule.Directive{
				{Key: "scala_debug", Value: "imports"},
				{Key: "scala_rule", Value: "scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary"},
			},
			want: &Config{
				rules: map[string]*scalarule.Config{
					"scala_binary": {
						Deps:           map[string]bool{},
						Attrs:          map[string]map[string]bool{},
						Options:        map[string]bool{},
						Enabled:        true,
						Implementation: "@io_bazel_rules_scala//scala:scala.bzl%scala_binary",
						Name:           "scala_binary",
					},
				},
				annotations: map[debugAnnotation]any{
					DebugImports: nil,
				},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
			},
		},
		"debug deps": {
			directives: []rule.Directive{
				{Key: "scala_debug", Value: "deps"},
			},
			want: &Config{
				rules:             map[string]*scalarule.Config{},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
				annotations: map[debugAnnotation]any{
					DebugDeps: nil,
				},
			},
		},
		"debug imports": {
			directives: []rule.Directive{
				{Key: "scala_debug", Value: "imports"},
			},
			want: &Config{
				rules:             map[string]*scalarule.Config{},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
				annotations: map[debugAnnotation]any{
					DebugImports: nil,
				},
			},
		},
		"scala_generate_build_files": {
			directives: []rule.Directive{
				{Key: "scala_generate_build_files", Value: "true"},
			},
			want: &Config{
				rules:              map[string]*scalarule.Config{},
				labelNameRewrites:  map[string]resolver.LabelNameRewriteSpec{},
				annotations:        map[debugAnnotation]interface{}{},
				generateBuildFiles: true,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc
			if diff := cmp.Diff(tc.want, got,
				cmp.AllowUnexported(Config{}),
				cmpopts.IgnoreFields(Config{}, "config", "universe", "logger"),
				cmpopts.IgnoreFields(scalarule.Config{}, "Config", "Logger"),
			); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigShouldAnnotateImports(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		want       bool
	}{
		"degenerate": {
			want: false,
		},
		"false": {
			directives: []rule.Directive{
				{Key: "scala_debug", Value: "-imports"},
			},
			want: false,
		},
		"true": {
			directives: []rule.Directive{
				{Key: "scala_debug", Value: "imports"},
			},
			want: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if err != nil {
				t.Fatal(err)
			}
			got := sc.ShouldAnnotateImports()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseRuleDirective(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       map[string]*scalarule.Config
	}{
		"degenerate": {
			want: map[string]*scalarule.Config{},
		},
		"bad format": {
			directives: []rule.Directive{
				{Key: scalaRuleDirective, Value: "myrule existing_scala_library_rule"},
			},
			wantErr: fmt.Errorf(`invalid directive: "gazelle:scala_rule myrule existing_scala_library_rule": expected three or more fields, got 2`),
		},
		"example": {
			directives: []rule.Directive{
				{Key: scalaRuleDirective, Value: "myrule implementation existing_scala_library_rule"},
				{Key: scalaRuleDirective, Value: "myrule deps @maven//:a"},
				{Key: scalaRuleDirective, Value: "myrule +deps @maven//:b"},
				{Key: scalaRuleDirective, Value: "myrule -deps @maven//:c"},
				{Key: scalaRuleDirective, Value: "myrule attr exports @maven//:a"},
				{Key: scalaRuleDirective, Value: "myrule option -fake_flag_name fake_flag_value"},
				{Key: scalaRuleDirective, Value: "myrule enabled false"},
			},
			want: map[string]*scalarule.Config{
				"myrule": {
					Config:         config.New(),
					Name:           "myrule",
					Implementation: "existing_scala_library_rule",
					Deps: map[string]bool{
						"@maven//:a": true,
						"@maven//:b": true,
					},
					Attrs: map[string]map[string]bool{
						"exports": {"@maven//:a": true},
					},
					Options: map[string]bool{"-fake_flag_name fake_flag_value": true},
					Enabled: false,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.rules
			if diff := cmp.Diff(tc.want, got,
				cmpopts.IgnoreFields(scalarule.Config{}, "Config", "Logger"),
			); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseOverrideDirective(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       []*overrideSpec
	}{
		"degenerate": {},
		"scala scala": {
			directives: []rule.Directive{
				{Key: resolveGlobDirective, Value: "scala scala com.foo.Bar //com/foo/bar"},
			},
			want: []*overrideSpec{
				{
					imp:  resolve.ImportSpec{Lang: "scala", Imp: "com.foo.Bar"},
					lang: "scala",
					dep:  label.New("", "com/foo/bar", "bar"),
				},
			},
		},
		"scala glob": {
			directives: []rule.Directive{
				{Key: resolveGlobDirective, Value: "scala glob com.foo.* //com/foo/bar"},
			},
			want: []*overrideSpec{
				{
					imp:  resolve.ImportSpec{Lang: "scala", Imp: "com.foo.*"},
					lang: "glob",
					dep:  label.New("", "com/foo/bar", "bar"),
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.overrides
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(overrideSpec{})); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseImplicitImportDirective(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		want       []*implicitImportSpec
		wantErr    error
	}{
		"degenerate": {},
		"typical example": {
			directives: []rule.Directive{
				{Key: resolveWithDirective, Value: "java com.typesafe.scalalogging.LazyLogging org.slf4j.Logger"},
			},
			want: []*implicitImportSpec{
				{
					lang: "java",
					imp:  "com.typesafe.scalalogging.LazyLogging",
					deps: []string{"org.slf4j.Logger"},
				},
			},
		},
		"anatomic example": {
			directives: []rule.Directive{
				{Key: resolveWithDirective, Value: "lang imp a b c"},
			},
			want: []*implicitImportSpec{
				{
					lang: "lang",
					imp:  "imp",
					deps: []string{"a", "b", "c"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.implicitImports
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(implicitImportSpec{})); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseResolveFileSymbolName(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		filename   string
		names      []string
		want       []bool
		wantErr    error
	}{
		"degenerate": {
			want: []bool{},
		},
		"exact matches": {
			directives: []rule.Directive{
				{Key: resolveFileSymbolName, Value: "filename.scala +foo -bar"},
			},
			filename: "filename.scala",
			names:    []string{"foo", "bar"},
			want:     []bool{true, false},
		},
		"glob matches": {
			directives: []rule.Directive{
				{Key: resolveFileSymbolName, Value: "*.scala +foo* -bar*"},
			},
			filename: "filename.scala",
			names:    []string{"foo", "foox", "bar", "barx"},
			want:     []bool{true, true, false, false},
		},
		"no match": {
			directives: []rule.Directive{
				{Key: resolveFileSymbolName, Value: "*.scala +foo* -bar*"},
			},
			filename: "filename.scala",
			names:    []string{"baz"},
			want:     []bool{false},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := []bool{}
			for _, name := range tc.names {
				got = append(got, sc.ShouldResolveFileSymbolName(tc.filename, name))
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
func TestScalaConfigParseFixWildcardImports(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		filename   string
		imports    []string
		want       []bool
		wantErr    error
	}{
		"degenerate": {
			want: []bool{},
		},
		"exact matches": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "filename.scala omnistac.core.entity._"},
			},
			filename: "filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{true},
		},
		"glob matches": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "*.scala omnistac.core.entity._"},
			},
			filename: "filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{true},
		},
		"recursive glob matches non-recursive path": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "**/*.scala omnistac.core.entity._"},
			},
			filename: "filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{true},
		},
		"recursive glob matches": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "**/*.scala omnistac.core.entity._"},
			},
			filename: "path/to/filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{true},
		},
		"recursive glob matches only absolute path (absolute version)": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "/**/*.scala omnistac.core.entity._"},
			},
			filename: "path/to/filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{false},
		},
		"recursive glob matches absolute path (absolute version)": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "/**/*.scala omnistac.core.entity._"},
			},
			filename: "/path/to/filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{true},
		},
		"no match": {
			directives: []rule.Directive{
				{Key: scalaFixWildcardImportDirective, Value: "*.scala -omnistac.core.entity._"},
			},
			filename: "filename.scala",
			imports:  []string{"omnistac.core.entity._"},
			want:     []bool{false},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := []bool{}
			for _, imp := range tc.imports {
				got = append(got, sc.ShouldFixWildcardImport(tc.filename, imp))
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseScalaAnnotate(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       map[debugAnnotation]interface{}
	}{
		"degenerate": {
			want: map[debugAnnotation]interface{}{},
		},
		"imports": {
			directives: []rule.Directive{
				{Key: scalaDebugDirective, Value: "imports"},
			},
			want: map[debugAnnotation]interface{}{
				DebugImports: nil,
			},
		},
		"exports": {
			directives: []rule.Directive{
				{Key: scalaDebugDirective, Value: "exports"},
			},
			want: map[debugAnnotation]interface{}{
				DebugExports: nil,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.annotations
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseResolveKindRewriteNameDirective(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       map[string]resolver.LabelNameRewriteSpec
	}{
		"degenerate": {
			want: map[string]resolver.LabelNameRewriteSpec{},
		},
		"anatomic example": {
			directives: []rule.Directive{
				{Key: resolveKindRewriteNameDirective, Value: "kind src dst"},
			},
			want: map[string]resolver.LabelNameRewriteSpec{
				"kind": {Src: "src", Dst: "dst"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := NewTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.labelNameRewrites
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigGetKnownRule(t *testing.T) {
	for name, tc := range map[string]struct {
		repoName  string
		rel       string
		from      label.Label
		want      label.Label
		wantTimes int
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
			rel:       "a",
			from:      label.New("", "", "test"),
			want:      label.New("", "a", "test"),
			wantTimes: 1,
		},
		"absolute remains unchanged": {
			from:      label.New("bar", "b", "test"),
			want:      label.New("bar", "b", "test"),
			wantTimes: 1,
		},
		"absolute without package unchanged": {
			repoName:  "foo",
			rel:       "a",
			from:      label.New("bar", "", "test"),
			want:      label.New("bar", "", "test"),
			wantTimes: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config.New()
			c.RepoName = tc.repoName
			universe := mocks.NewUniverse(t)

			var got label.Label
			capture := func(from label.Label) bool {
				got = from
				return true
			}
			universe.
				On("GetKnownRule", mock.MatchedBy(capture)).
				Maybe().
				Times(tc.wantTimes).
				Return(nil, false)

			sc := New(zerolog.New(os.Stderr), universe, c, tc.rel)

			sc.GetKnownRule(tc.from)

			universe.AssertExpectations(t)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
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
			got := labelFromDepExpr(expr)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("label (-want +got):\n%s", diff)
			}
		})
	}
}
