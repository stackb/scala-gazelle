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
		want      label.Label
		wantOk    bool
	}{
		"degenerate": {
			symbol: resolver.Symbol{
				Name:  "org.json4s",
				Label: label.Label{Repo: "maven", Pkg: "", Name: "org_json4s_json4s_core_2_13"},
			},
			wantOk: false,
		},
		"should not select preferred label when no preferred map": {
			symbol: resolver.Symbol{
				Name:  "org.json4s",
				Label: label.Label{Repo: "maven", Pkg: "", Name: "org_json4s_json4s_core_2_13"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "org.json4s",
						Label: label.Label{Repo: "maven", Pkg: "", Name: "org_json4s_json4s_ast_2_13"},
					},
				},
			},
			wantOk: false,
		},
		"should select preferred label when preferred map matches name (parent case)": {
			preferred: map[string]label.Label{
				"org.json4s": {Repo: "maven", Pkg: "", Name: "org_json4s_json4s_core_2_13"},
			},
			symbol: resolver.Symbol{
				Name:  "org.json4s",
				Label: mustParseLabel(t, "@maven//:org_json4s_json4s_core_2_13"),
				Conflicts: []*resolver.Symbol{
					{
						Name:  "org.json4s",
						Label: mustParseLabel(t, "@maven//:org_json4s_json4s_ast_2_13"),
					},
				},
			},
			wantOk: true,
			want:   mustParseLabel(t, "@maven//:org_json4s_json4s_core_2_13"),
		},
		"should select preferred label when preferred map matches name (child case)": {
			preferred: map[string]label.Label{
				"org.json4s": {Repo: "maven", Pkg: "", Name: "org_json4s_json4s_ast_2_13"},
			},
			symbol: resolver.Symbol{
				Name:  "org.json4s",
				Label: mustParseLabel(t, "@maven//:org_json4s_json4s_core_2_13"),
				Conflicts: []*resolver.Symbol{
					{
						Name:  "org.json4s",
						Label: mustParseLabel(t, "@maven//:org_json4s_json4s_ast_2_13"),
					},
				},
			},
			wantOk: true,
			want:   mustParseLabel(t, "@maven//:org_json4s_json4s_ast_2_13"),
		},
		"should not select preferred label when preferred map matches none": {
			preferred: map[string]label.Label{
				"org.json4s": {Repo: "maven", Pkg: "", Name: "org_json4s_json4s_full_2_13"},
			},
			symbol: resolver.Symbol{
				Name:  "org.json4s",
				Label: mustParseLabel(t, "@maven//:org_json4s_json4s_core_2_13"),
				Conflicts: []*resolver.Symbol{
					{
						Name:  "org.json4s",
						Label: mustParseLabel(t, "@maven//:org_json4s_json4s_ast_2_13"),
					},
				},
			},
			wantOk: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := mocks.NewUniverse(t)
			resolver := resolver.NewPreferredDepsConflictResolver("preferred_deps", tc.preferred)
			gotSymbol, gotOk := resolver.ResolveConflict(universe, &tc.rule, tc.imports, &tc.imp, &tc.symbol)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ok (-want +got):\n%s", diff)
			}
			got := label.NoLabel
			if gotSymbol != nil {
				got = gotSymbol.Label
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseLabel(t *testing.T, s string) label.Label {
	from, err := label.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return from
}
