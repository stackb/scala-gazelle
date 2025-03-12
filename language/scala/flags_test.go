package scala

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestCacheFlags(t *testing.T) {
	for name, tc := range map[string]struct {
		args    []string
		files   []testtools.FileSpec
		wantErr error
		check   func(t *testing.T, tmpDir string, lang *scalaLang)
	}{
		"scala_gazelle_cache_file": {
			files: []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: "{}",
				},
				{
					Path:    "./cache.json",
					Content: `{"package_count": 100}`,
				},
			},
			args: []string{
				"-maven_install_json_file=./maven_install.json",
				"-scala_gazelle_cache_file=${TEST_TMPDIR}/cache.json",
			},
			check: func(t *testing.T, tmpDir string, lang *scalaLang) {
				cacheFile := strings.TrimPrefix(strings.TrimPrefix(lang.cacheFileFlagValue, tmpDir), "/")
				if diff := cmp.Diff("cache.json", cacheFile); diff != "" {
					t.Errorf("cacheFile (-want got):\n%s", diff)
				}
				if diff := cmp.Diff(int32(100), lang.cache.PackageCount); diff != "" {
					t.Errorf("PackageCount (-want got):\n%s", diff)
				}
			},
		},
		"scala_gazelle_print_cache_key_on": {
			args: []string{
				"-scala_gazelle_cache_key=12345",
				"-scala_gazelle_print_cache_key=true",
			},
			check: func(t *testing.T, tmpDir string, lang *scalaLang) {
				if diff := cmp.Diff("12345", lang.cacheKeyFlagValue); diff != "" {
					t.Errorf("cacheKeyFlagValue (-want got):\n%s", diff)
				}
				if diff := cmp.Diff(true, lang.printCacheKey); diff != "" {
					t.Errorf("printCacheKey (-want got):\n%s", diff)
				}
			},
		},
		"scala_gazelle_print_cache_key_off": {
			args: []string{
				"-scala_gazelle_print_cache_key=false",
			},
			check: func(t *testing.T, tmpDir string, lang *scalaLang) {
				if diff := cmp.Diff(false, lang.printCacheKey); diff != "" {
					t.Errorf("printCacheKey (-want got):\n%s", diff)
				}
			},
		},
		"scala_gazelle_cache_key__valid": {
			files: []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: "{}",
				},
				{
					Path:    "./cache.json",
					Content: `{"package_count": 100, "key": "1"}`,
				},
			},
			args: []string{
				"-maven_install_json_file=./maven_install.json",
				"-scala_gazelle_cache_file=${TEST_TMPDIR}/cache.json",
				"-scala_gazelle_cache_key=1",
			},
			check: func(t *testing.T, tmpDir string, lang *scalaLang) {
				if lang.cache.PackageCount == 0 {
					t.Error("expected cache to be valid!")
				}
			},
		},
		"scala_gazelle_cache_key__invalid": {
			files: []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: "{}",
				},
				{
					Path:    "./cache.json",
					Content: `{"package_count": 100, "key": "1"}`,
				},
			},
			args: []string{
				"-maven_install_json_file=./maven_install.json",
				"-scala_gazelle_cache_file=${TEST_TMPDIR}/cache.json",
				"-scala_gazelle_cache_key=2",
			},
			check: func(t *testing.T, tmpDir string, lang *scalaLang) {
				if lang.cache.PackageCount != 0 {
					t.Error("expected cache to be invalidated!")
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustPrepareTestFiles(t, tc.files)
			if true {
				defer cleanup()
			}

			os.Setenv("TEST_TMPDIR", tmpDir)
			lang := NewLanguage().(*scalaLang)

			fs := flag.NewFlagSet(scalaLangName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
				Exts:    make(map[string]interface{}),
			}

			lang.RegisterFlags(fs, "", c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			if err := lang.CheckFlags(fs, c); err != nil {
				t.Fatal(err)
			}

			if tc.check != nil {
				tc.check(t, tmpDir, lang)
			}
		})
	}
}

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
