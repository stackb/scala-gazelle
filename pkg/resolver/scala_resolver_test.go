package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stretchr/testify/mock"
)

func TestScalaResolver(t *testing.T) {
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
	} {
		t.Run(name, func(t *testing.T) {
			var got string
			captureImport := func(imp string) bool {
				got = imp
				return true
			}
			importResolver := mocks.NewKnownImportResolver(t)
			importResolver.
				On("ResolveKnownImport",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.MatchedBy(captureImport),
				).
				Maybe().
				Return(nil, nil)

			rslv := resolver.NewScalaResolver(importResolver)
			c := config.New()

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)

			rslv.ResolveKnownImport(c, ix, tc.from, tc.lang, tc.imp)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
