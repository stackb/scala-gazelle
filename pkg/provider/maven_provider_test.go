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
	"github.com/stretchr/testify/mock"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

const scalaName = "scala"

func ExampleMavenProvider_RegisterFlags_printdefaults() {
	os.Stderr = os.Stdout
	cr := provider.NewMavenProvider(scalaName)
	got := flag.NewFlagSet(scalaName, flag.ExitOnError)
	c := &config.Config{}
	cr.RegisterFlags(got, "update", c)
	got.PrintDefaults()
	// output:
	//	-maven_install_json_file value
	//     	path to maven_install.json file
}

func TestMavenProviderFlags(t *testing.T) {
	for name, tc := range map[string]struct {
		args  []string
		files []testtools.FileSpec
		want  []*resolver.Symbol
	}{
		"empty maven file": {
			args: []string{
				"-maven_install_json_file=./maven_install.json",
			},
			files: []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: "{}",
				},
			},
			want: nil,
		},
		"example maven file": {
			args: []string{
				"-maven_install_json_file=./testdata/maven_install.json",
			},
			files: []testtools.FileSpec{
				{
					Path: "testdata/maven_install.json",
				},
			},
			want: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "javax.xml",
					Label:    label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
					Provider: "maven",
				},
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "javax.xml.datatype",
					Label:    label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
					Provider: "maven",
				},
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "javax.xml.namespace",
					Label:    label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
					Provider: "maven",
				},
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "javax.xml.parsers",
					Label:    label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
					Provider: "maven",
				},
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "javax.xml.stream",
					Label:    label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
					Provider: "maven",
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustReadAndPrepareTestFiles(t, tc.files)
			defer cleanup()

			p := provider.NewMavenProvider(scalaName)
			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
			}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			scope := mocks.NewScope(t)
			var got []*resolver.Symbol
			capture := func(known *resolver.Symbol) bool {
				got = append(got, known)
				return true
			}
			scope.
				On("PutSymbol", mock.MatchedBy(capture)).
				Maybe().
				Times(len(tc.want)).
				Return(nil)

			if err := p.CheckFlags(fs, c, scope); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".mavenInstallFile (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMavenProviderCanProvide(t *testing.T) {
	for name, tc := range map[string]struct {
		lang string
		from label.Label
		expr build.Expr
		want bool
	}{
		"degenerate case": {
			lang: scalaName,
			from: label.NoLabel,
			want: false,
		},
		"managed xml_apis_xml_apis": {
			lang: scalaName,
			from: label.New("maven", "", "xml_apis_xml_apis"),
			want: true,
		},
		"managed generic maven dependency": {
			lang: scalaName,
			from: label.New("maven", "", "com_guava_guava"),
			want: true,
		},
		"unmanaged non-maven dependency": {
			lang: scalaName,
			from: label.New("artifactory", "", "xml_apis_xml_apis"),
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustReadAndPrepareTestFiles(t, []testtools.FileSpec{
				{Path: "testdata/maven_install.json"},
			})
			defer cleanup()

			p := provider.NewMavenProvider(scalaName)

			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{WorkDir: tmpDir}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse([]string{
				"-maven_install_json_file=./testdata/maven_install.json",
			}); err != nil {
				t.Fatal(err)
			}

			scope := mocks.NewScope(t)
			scope.On("PutSymbol", mock.Anything).Maybe().Return(nil)

			if err := p.CheckFlags(fs, c, scope); err != nil {
				t.Fatal(err)
			}

			got := p.CanProvide(&resolver.ImportLabel{Label: tc.from}, tc.expr, func(from label.Label) (*rule.Rule, bool) {
				return nil, false
			}, tc.from)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".CanProvide (-want +got):\n%s", diff)
			}
		})
	}
}
