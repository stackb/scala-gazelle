package scala

import (
	"fmt"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestParseScalaExistingRules(t *testing.T) {
	for name, tc := range map[string]struct {
		providerNames []string
		wantErr       error
		wantLoadInfo  rule.LoadInfo
		wantKindInfo  rule.KindInfo
		check         func(t *testing.T)
	}{
		"degenerate": {},
		"invalid flag value": {
			providerNames: []string{"@io_bazel_rules_scala//scala:scala.bzl#scala_binary"},
			wantErr:       fmt.Errorf(`invalid -existing_scala_binary_rule flag value: wanted '%%' separated string, got "@io_bazel_rules_scala//scala:scala.bzl#scala_binary"`),
		},
		"valid flag value": {
			providerNames: []string{"//custom/scala:scala.bzl%scala_binary"},
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
			lang := NewLanguage().(*scalaLang)
			lang.ruleProviderRegistry = scalarule.NewProviderRegistryMap() // don't use global one
			if testutil.ExpectError(t, tc.wantErr, lang.setupExistingScalaBinaryRules(tc.providerNames)) {
				return
			}
			if tc.check != nil {
				tc.check(t)
			}
			for _, name := range tc.providerNames {
				if provider, ok := lang.ruleProviderRegistry.LookupProvider(name); ok {
					if diff := cmp.Diff(tc.wantLoadInfo, provider.LoadInfo()); diff != "" {
						t.Errorf("loadInfo (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(tc.wantKindInfo, provider.KindInfo()); diff != "" {
						t.Errorf("kindInfo (-want +got):\n%s", diff)
					}
				} else {
					t.Fatal("unexpected false value for ")
				}
			}
		})
	}
}
