package semanticdb

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

// TestSemanticdbIndexRuleResolveDeps asserts that index membership is
// independent of symbol insertion order: TrieScope.Put is first-wins for a
// given symbol name, so a label whose every symbol name is already claimed
// by another label only appears in the scope as a Conflict.  Membership must
// include conflict losers, otherwise the generated deps differ between runs
// that insert symbols in package-walk order and runs that preload them from
// a cache in sorted order.
func TestSemanticdbIndexRuleResolveDeps(t *testing.T) {
	for name, tc := range map[string]struct {
		kinds   []string
		symbols []*resolver.Symbol
		want    []string
	}{
		"conflict loser is a member": {
			kinds: []string{"scala_library"},
			symbols: []*resolver.Symbol{
				resolver.NewSymbol(sppb.ImportType_CLASS, "com.foo.Dup", "scala_library", label.New("", "a", "a")),
				resolver.NewSymbol(sppb.ImportType_CLASS, "com.foo.Dup", "scala_library", label.New("", "b", "b")),
			},
			want: []string{"//a:a_semanticdb", "//b:b_semanticdb"},
		},
		"kind filter still applies to conflict losers": {
			kinds: []string{"scala_library"},
			symbols: []*resolver.Symbol{
				resolver.NewSymbol(sppb.ImportType_CLASS, "com.foo.Dup", "scala_library", label.New("", "a", "a")),
				resolver.NewSymbol(sppb.ImportType_CLASS, "com.foo.Dup", "scala_binary", label.New("", "b", "b")),
			},
			want: []string{"//a:a_semanticdb"},
		},
		"external labels are excluded": {
			kinds: []string{"scala_library"},
			symbols: []*resolver.Symbol{
				resolver.NewSymbol(sppb.ImportType_CLASS, "com.foo.Dup", "scala_library", label.New("", "a", "a")),
				resolver.NewSymbol(sppb.ImportType_CLASS, "com.foo.Dup", "scala_library", label.New("maven", "", "jar")),
			},
			want: []string{"//a:a_semanticdb"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := resolver.NewTrieScope()
			for _, sym := range tc.symbols {
				if err := scope.PutSymbol(sym); err != nil {
					t.Fatal(err)
				}
			}
			SetGlobalScope(scope)

			r := rule.NewRule(SemanticdbIndexRuleKind, "semanticdb_index")
			r.SetAttr("kinds", tc.kinds)

			provider := NewSemanticdbIndexRuleProvider(SemanticdbIndexRuleLoad, SemanticdbIndexRuleKind)
			ruleProvider := provider.ResolveRule(nil, nil, r)
			ruleProvider.Resolve(&scalarule.ResolveContext{Rule: r}, nil)

			if diff := cmp.Diff(tc.want, r.AttrStrings("deps")); diff != "" {
				t.Fatalf("deps (-want +got):\n%s", diff)
			}
		})
	}
}
