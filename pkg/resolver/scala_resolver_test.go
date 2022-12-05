package resolver

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
)

func TestScalaResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		lang    string
		from    label.Label
		known   []*KnownImport
		imports []*Import
		want    []*Import
		wantErr error
	}{
		"degenerate": {},
	} {
		t.Run(name, func(t *testing.T) {
			importRegistry := NewKnownImportRegistryTrie()
			for _, known := range tc.known {
				if err := importRegistry.PutKnownImport(known); err != nil {
					t.Fatal(err)
				}
			}

			rslv := NewScalaResolver("scala", importRegistry)
			c := config.New()

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)

			rslv.ResolveImports(c, ix, tc.from, tc.lang, tc.imports...)

			if diff := cmp.Diff(tc.want, tc.imports); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
