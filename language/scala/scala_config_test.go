package scala

import (
	"fmt"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/mock"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestScalaConfigParseDirectives(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       *scalaConfig
	}{
		"degenerate": {
			want: &scalaConfig{
				rules:             map[string]*scalarule.Config{},
				annotations:       map[annotation]any{},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
			},
		},
		"annotation after rule": {
			directives: []rule.Directive{
				{Key: "scala_rule", Value: "scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary"},
				{Key: "scala_annotate", Value: "imports"},
			},
			want: &scalaConfig{
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
				annotations: map[annotation]any{
					AnnotateImports: nil,
				},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
			},
		},
		"rule after annotation": {
			directives: []rule.Directive{
				{Key: "scala_annotate", Value: "imports"},
				{Key: "scala_rule", Value: "scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary"},
			},
			want: &scalaConfig{
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
				annotations: map[annotation]any{
					AnnotateImports: nil,
				},
				labelNameRewrites: map[string]resolver.LabelNameRewriteSpec{},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := newTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc
			if diff := cmp.Diff(tc.want, got,
				cmp.AllowUnexported(scalaConfig{}),
				cmpopts.IgnoreFields(scalaConfig{}, "config", "universe"),
				cmpopts.IgnoreFields(scalarule.Config{}, "Config"),
			); diff != "" {
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
				{Key: ruleDirective, Value: "myrule existing_scala_rule"},
			},
			wantErr: fmt.Errorf(`invalid directive: "gazelle:scala_rule myrule existing_scala_rule": expected three or more fields, got 2`),
		},
		"example": {
			directives: []rule.Directive{
				{Key: ruleDirective, Value: "myrule implementation existing_scala_rule"},
				{Key: ruleDirective, Value: "myrule deps @maven//:a"},
				{Key: ruleDirective, Value: "myrule +deps @maven//:b"},
				{Key: ruleDirective, Value: "myrule -deps @maven//:c"},
				{Key: ruleDirective, Value: "myrule attr exports @maven//:a"},
				{Key: ruleDirective, Value: "myrule option -fake_flag_name fake_flag_value"},
				{Key: ruleDirective, Value: "myrule enabled false"},
			},
			want: map[string]*scalarule.Config{
				"myrule": {
					Config:         config.New(),
					Name:           "myrule",
					Implementation: "existing_scala_rule",
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
			sc, err := newTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.rules
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
			sc, err := newTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
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
			sc, err := newTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
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

func TestScalaConfigParseScalaAnnotate(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       map[annotation]interface{}
	}{
		"degenerate": {
			want: map[annotation]interface{}{},
		},
		"imports": {
			directives: []rule.Directive{
				{Key: scalaAnnotateDirective, Value: "imports"},
			},
			want: map[annotation]interface{}{
				AnnotateImports: nil,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := newTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
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
				{Key: resolveKindRewriteName, Value: "kind src dst"},
			},
			want: map[string]resolver.LabelNameRewriteSpec{
				"kind": {Src: "src", Dst: "dst"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := newTestScalaConfig(t, mocks.NewUniverse(t), "", tc.directives...)
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
		"makes repo absolute": {
			repoName:  "foo",
			from:      label.New("", "", "test"),
			want:      label.New("foo", "", "test"),
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

			sc := newScalaConfig(universe, c, tc.rel)

			sc.GetKnownRule(tc.from)

			universe.AssertExpectations(t)
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
			sc := newScalaConfig(mocks.NewUniverse(t), c, "")
			got := isSameImport(sc, tc.kind, tc.from, tc.to)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func newTestScalaConfig(t *testing.T, universe resolver.Universe, rel string, dd ...rule.Directive) (*scalaConfig, error) {
	c := config.New()
	sc := newScalaConfig(universe, c, rel)
	err := sc.parseDirectives(dd)
	return sc, err
}
