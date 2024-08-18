package provider_test

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

var gsonLabel = label.Label{Repo: "maven", Name: "com_google_code_gson_gson"}

func ExampleJavaProvider_RegisterFlags_printdefaults() {
	os.Stderr = os.Stdout
	cr := provider.NewJavaProvider()
	got := flag.NewFlagSet("", flag.ExitOnError)
	c := &config.Config{}
	cr.RegisterFlags(got, "update", c)
	got.PrintDefaults()
	// output:
	//	-javaindex_file value
	//     	path to javaindex.pb or javaindex.json file
}

func TestJavaProviderOnResolve(t *testing.T) {
	for name, tc := range map[string]struct {
		files []testtools.FileSpec
		known []*resolver.Symbol
		want  []*resolver.Symbol
	}{
		"empty file": {
			files: []testtools.FileSpec{
				{Path: "testdata/javaindex.json", Content: "{}"},
			},
		},
		"example java_index file": {
			files: []testtools.FileSpec{
				{Path: "testdata/javaindex.json"},
			},
			known: []*resolver.Symbol{
				{Type: sppb.ImportType_CLASS, Name: "java.lang.Enum", Label: label.NoLabel, Provider: "java"},
				{Type: sppb.ImportType_CLASS, Name: "java.lang.Iterable", Label: label.NoLabel, Provider: "java"},
				{Type: sppb.ImportType_CLASS, Name: "java.lang.RuntimeException", Label: label.NoLabel, Provider: "java"},
			},
			want: []*resolver.Symbol{
				{Type: sppb.ImportType_PACKAGE, Name: "com.google.gson", Label: gsonLabel, Provider: "java"},
				{Type: sppb.ImportType_INTERFACE, Name: "com.google.gson.ExclusionStrategy", Label: gsonLabel, Provider: "java"},
				{Type: sppb.ImportType_CLASS, Name: "com.google.gson.FieldAttributes", Label: gsonLabel, Provider: "java"},
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "com.google.gson.FieldNamingPolicy",
					Label:    gsonLabel,
					Provider: "java",
					Requires: []*resolver.Symbol{
						{Type: sppb.ImportType_CLASS, Name: "java.lang.Enum", Provider: "java"},
						{
							Type:     sppb.ImportType_INTERFACE,
							Name:     "com.google.gson.FieldNamingStrategy",
							Label:    gsonLabel,
							Provider: "java",
						},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustReadAndPrepareTestFiles(t, tc.files)
			defer cleanup()

			p := provider.NewJavaProvider()
			fs := flag.NewFlagSet("", flag.ExitOnError)
			c := &config.Config{WorkDir: tmpDir}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse([]string{
				"-javaindex_file=./testdata/javaindex.json",
			}); err != nil {
				t.Fatal(err)
			}

			scope := resolver.NewTrieScope()
			for _, known := range tc.known {
				scope.PutSymbol(known)
			}

			if err := p.CheckFlags(fs, c, scope); err != nil {
				t.Fatal(err)
			}

			p.OnResolve()

			got := scope.Symbols()
			// don't need to be exhaustive here, just look at the first few...
			if len(got) > 4 {
				got = got[:4]
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestJavaProviderCanProvide(t *testing.T) {
	for name, tc := range map[string]struct {
		from label.Label
		expr build.Expr
		want bool
	}{
		"no label": {
			from: label.NoLabel,
			want: false,
		},
		"not in the index": {
			from: label.Label{Repo: "@other_maven", Name: "other_lib"},
			want: false,
		},
		"in the index": {
			from: gsonLabel,
			want: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustReadAndPrepareTestFiles(t, []testtools.FileSpec{
				{Path: "testdata/javaindex.json"},
			})
			defer cleanup()

			p := provider.NewJavaProvider()
			fs := flag.NewFlagSet("", flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
			}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse([]string{
				"-javaindex_file=./testdata/javaindex.json",
			}); err != nil {
				t.Fatal(err)
			}

			scope := resolver.NewTrieScope()

			if err := p.CheckFlags(fs, c, scope); err != nil {
				t.Fatal(err)
			}
			p.OnResolve()

			got := p.CanProvide(&resolver.ImportLabel{Label: tc.from}, tc.expr, func(from label.Label) (*rule.Rule, bool) {
				return nil, false
			})

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
