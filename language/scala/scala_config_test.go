package scala

import (
	"fmt"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
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
			if !equalError(tc.wantErr, err) {
				t.Fatal("errors: want:", tc.wantErr, "got:", err)
			}
			if tc.wantErr != nil {
				return
			}
			got := sc.rules
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
