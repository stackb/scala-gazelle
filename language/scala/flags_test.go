package scala

import (
	"fmt"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
)

func TestParseScalaExistingRules(t *testing.T) {
	for name, tc := range map[string]struct {
		rules        []string
		wantErr      error
		wantLoadInfo rule.LoadInfo
		wantKindInfo rule.KindInfo
		check        func(t *testing.T)
	}{
		"degenerate": {},
		"invalid flag value": {
			rules:   []string{"@io_bazel_rules_scala//scala:scala.bzl#scala_binary"},
			wantErr: fmt.Errorf(`invalid -scala_existing_rule flag value: wanted '%%' separated string, got "@io_bazel_rules_scala//scala:scala.bzl#scala_binary"`),
		},
		"valid flag value": {
			rules: []string{"//custom/scala:scala.bzl%scala_binary"},
			wantLoadInfo: rule.LoadInfo{
				Name:    "//custom/scala:scala.bzl",
				Symbols: []string{"scala_binary"},
			},
			wantKindInfo: rule.KindInfo{
				ResolveAttrs: map[string]bool{"deps": true},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := parseScalaExistingRules(tc.rules)
			if !equalError(tc.wantErr, err) {
				t.Fatal("errors: want:", tc.wantErr, "got:", err)
			}
			if tc.wantErr != nil {
				return
			}
			if tc.check != nil {
				tc.check(t)
			}
			for _, ruleID := range tc.rules {
				if info, err := Rules().LookupRule(ruleID); err != nil {
					t.Fatal(err)
				} else {
					if diff := cmp.Diff(tc.wantLoadInfo, info.LoadInfo()); diff != "" {
						t.Errorf("loadInfo (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(tc.wantKindInfo, info.KindInfo()); diff != "" {
						t.Errorf("kindInfo (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

// equalError reports whether errors a and b are considered equal.
// They're equal if both are nil, or both are not nil and a.Error() == b.Error().
func equalError(a, b error) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Error() == b.Error()
}
