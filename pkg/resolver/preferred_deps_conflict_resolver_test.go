package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
)

func TestPreferredDepsConflictResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		preferred map[string]label.Label
		symbol    resolver.Symbol
		rule      rule.Rule
		imports   resolver.ImportMap
		imp       resolver.Import
		name      string
		want      *resolver.Symbol
		wantOk    bool
	}{
		"degenerate": {
			symbol: resolver.Symbol{
				Name:  "foo.Bar",
				Label: label.Label{Pkg: "foo", Name: "bar"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := mocks.NewUniverse(t)
			resolver := resolver.NewPreferredDepsConflictResolver("preferred_deps", tc.preferred)
			got, gotOk := resolver.ResolveConflict(universe, &tc.rule, tc.imports, &tc.imp, &tc.symbol)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ok (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
