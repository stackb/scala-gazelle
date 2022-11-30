package scala

import (
	"fmt"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestScalaConfigParseRuleDirective(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       map[string]*RuleConfig
	}{
		"degenerate": {
			want: map[string]*RuleConfig{},
		},
		"bad format": {
			directives: []rule.Directive{
				{Key: ruleDirective, Value: "myrule scala_existing_rule"},
			},
			wantErr: fmt.Errorf(`invalid directive: "gazelle:scala_rule myrule scala_existing_rule": expected three or more fields, got 2`),
		},
		"example": {
			directives: []rule.Directive{
				{Key: ruleDirective, Value: "myrule implementation scala_existing_rule"},
				{Key: ruleDirective, Value: "myrule deps @maven//:a"},
				{Key: ruleDirective, Value: "myrule +deps @maven//:b"},
				{Key: ruleDirective, Value: "myrule -deps @maven//:c"},
				{Key: ruleDirective, Value: "myrule attr exports @maven//:a"},
				{Key: ruleDirective, Value: "myrule option -fake_flag_name fake_flag_value"},
				{Key: ruleDirective, Value: "myrule enabled false"},
			},
			want: map[string]*RuleConfig{
				"myrule": {
					Config:         config.New(),
					Name:           "myrule",
					Implementation: "scala_existing_rule",
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
			sc, err := parseTestDirectives("", tc.directives...)
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
		"degenerate": {
			want: []*overrideSpec{},
		},
		"bad example - silently ignores everything other than scala glob": {
			directives: []rule.Directive{
				{Key: overrideDirective, Value: "scala scala com.foo.Bar //com/foo/bar"},
			},
			want: []*overrideSpec{},
		},
		"example - scala glob": {
			directives: []rule.Directive{
				{Key: overrideDirective, Value: "scala glob com.foo.* //com/foo/bar"},
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
			sc, err := parseTestDirectives("", tc.directives...)
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
		"degenerate": {
			want: []*implicitImportSpec{},
		},
		"typical example": {
			directives: []rule.Directive{
				{Key: implicitImportDirective, Value: "java com.typesafe.scalalogging.LazyLogging org.slf4j.Logger"},
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
				{Key: implicitImportDirective, Value: "lang imp a b c"},
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
			sc, err := parseTestDirectives("", tc.directives...)
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

func TestScalaConfigParseScalaExplainDependencies(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       bool
	}{
		"degenerate": {},
		"typical example": {
			directives: []rule.Directive{
				{Key: scalaExplainDependencies, Value: "true"},
			},
			want: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := parseTestDirectives("", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.explainDependencies
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaConfigParseMapKindImportNameDirective(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []rule.Directive
		wantErr    error
		want       map[string]mapKindImportNameSpec
	}{
		"degenerate": {
			want: map[string]mapKindImportNameSpec{},
		},
		"anatomic example": {
			directives: []rule.Directive{
				{Key: mapKindImportNameDirective, Value: "kind src dst"},
			},
			want: map[string]mapKindImportNameSpec{
				"kind": {src: "src", dst: "dst"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			sc, err := parseTestDirectives("", tc.directives...)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := sc.mapKindImportNames
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(mapKindImportNameSpec{})); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func parseTestDirectives(rel string, dd ...rule.Directive) (*scalaConfig, error) {
	index := &mockRuleIndex{}
	c := config.New()
	sc := newScalaConfig(index, c, rel)
	err := sc.parseDirectives(dd)
	return sc, err
}

func TestScalaConfigLookupRule(t *testing.T) {
	for name, tc := range map[string]struct {
		repoName string
		rel      string
		from     label.Label
		want     label.Label
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
			rel:  "a",
			from: label.New("", "", "test"),
			want: label.New("", "a", "test"),
		},
		"makes repo absolute": {
			repoName: "foo",
			from:     label.New("", "", "test"),
			want:     label.New("foo", "", "test"),
		},
		"absolute remains unchanged": {
			from: label.New("bar", "b", "test"),
			want: label.New("bar", "b", "test"),
		},
		"absolute without package unchanged": {
			repoName: "foo",
			rel:      "a",
			from:     label.New("bar", "", "test"),
			want:     label.New("bar", "", "test"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := config.New()
			c.RepoName = tc.repoName
			index := &mockRuleIndex{}
			sc := newScalaConfig(index, c, tc.rel)
			sc.LookupRule(tc.from)
			got := index.from
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
