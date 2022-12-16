package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
)

func TestScalaSymbolResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		lang string
		from label.Label
		imp  string
		want string
	}{
		"degenerate": {},
		"unchanged": {
			lang: "scala",
			from: label.Label{Pkg: "src", Name: "scala"},
			imp:  "com.foo.bar",
			want: "com.foo.bar",
		},
		"strips ^_root_.": {
			lang: "scala",
			from: label.Label{Pkg: "src", Name: "scala"},
			imp:  "_root_.scala.util.Try",
			want: "scala.util.Try",
		},
		"strips ._$": {
			lang: "scala",
			from: label.Label{Pkg: "src", Name: "scala"},
			imp:  "scala.util._",
			want: "scala.util",
		},
	} {
		t.Run(name, func(t *testing.T) {

			// FIXME(pcj): change this test to just assert what was resolved
			// rather than capturing the symbol passing to the internal resolver.

			var got string
			captureSymbol := func(sym string) bool {
				got = sym
				return true
			}
			symbolResolver := mocks.NewSymbolResolver(t)
			symbolResolver.
				On("ResolveSymbol",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.MatchedBy(captureSymbol),
				).
				Maybe().
				Return(nil, false)

			rslv := resolver.NewScalaSymbolResolver(symbolResolver)
			c := config.New()

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)

			rslv.ResolveSymbol(c, ix, tc.from, tc.lang, tc.imp)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
