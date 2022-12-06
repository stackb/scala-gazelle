package provider

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func ExampleJarIndexProvider_RegisterFlags_printdefaults() {
	os.Stderr = os.Stdout
	cr := NewJarIndexProvider()
	got := flag.NewFlagSet(scalaName, flag.ExitOnError)
	c := &config.Config{}
	cr.RegisterFlags(got, "update", c)
	got.PrintDefaults()
	// output:
	//	-jarindex_file value
	//     	path to jarindex.pb or jarindex.json file
}

func TestJarIndexProviderFlags(t *testing.T) {
	for name, tc := range map[string]struct {
		args  []string
		files []testtools.FileSpec
		want  []*resolver.KnownImport
	}{
		"empty file": {
			args: []string{
				"-jarindex_file=./jarindex.json",
			},
			files: []testtools.FileSpec{
				{
					Path:    "jarindex.json",
					Content: "{}",
				},
			},
			want: nil,
		},
		"example jarindex file": {
			args: []string{
				"-jarindex_file=./testdata/jarindex.json",
			},
			files: []testtools.FileSpec{
				{
					Path: "testdata/jarindex.json",
				},
			},
			want: []*resolver.KnownImport{
				// {
				// 	Type:   sppb.ImportType_PACKAGE,
				// 	Import: "javax.xml",
				// 	Label:  label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
				// },
				// {
				// 	Type:   sppb.ImportType_PACKAGE,
				// 	Import: "javax.xml.datatype",
				// 	Label:  label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
				// },
				// {
				// 	Type:   sppb.ImportType_PACKAGE,
				// 	Import: "javax.xml.namespace",
				// 	Label:  label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
				// },
				// {
				// 	Type:   sppb.ImportType_PACKAGE,
				// 	Import: "javax.xml.parsers",
				// 	Label:  label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
				// },
				// {
				// 	Type:   sppb.ImportType_PACKAGE,
				// 	Import: "javax.xml.stream",
				// 	Label:  label.Label{Repo: "maven", Name: "xml_apis_xml_apis"},
				// },
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustReadAndPrepareTestFiles(t, tc.files)
			defer cleanup()

			p := NewJarIndexProvider()
			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
			}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			importRegistry := &mockKnownImportRegistry{}

			if err := p.CheckFlags(fs, c, importRegistry); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, importRegistry.got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

// func TestJarIndexProviderCanProvide(t *testing.T) {
// 	for name, tc := range map[string]struct {
// 		mavenInstallJsonContent string
// 		lang                    string
// 		from                    label.Label
// 		want                    bool
// 	}{
// 		"degenerate case": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.NoLabel,
// 			want:                    false,
// 		},
// 		"managed xml_apis_xml_apis": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.New("maven", "", "xml_apis_xml_apis"),
// 			want:                    true,
// 		},
// 		"managed generic maven dependency": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.New("maven", "", "com_guava_guava"),
// 			want:                    true,
// 		},
// 		"unmanaged non-maven dependency": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.New("artifactory", "", "xml_apis_xml_apis"),
// 			want:                    false,
// 		},
// 	} {
// 		t.Run(name, func(t *testing.T) {
// 			tmpDir, _, cleanup := testutil.MustPrepareTestFiles(t, []testtools.FileSpec{
// 				{
// 					Path:    "jarindex.json",
// 					Content: tc.mavenInstallJsonContent,
// 				},
// 			})
// 			defer cleanup()

// 			p := NewJarIndexProvider(scalaName)
// 			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
// 			c := &config.Config{WorkDir: tmpDir}
// 			p.RegisterFlags(fs, "update", c)
// 			if err := fs.Parse([]string{
// 				"-jarindex_file=./jarindex.json",
// 			}); err != nil {
// 				t.Fatal(err)
// 			}

// 			importRegistry := &mockKnownImportRegistry{}

// 			if err := p.CheckFlags(fs, c, importRegistry); err != nil {
// 				t.Fatal(err)
// 			}

// 			got := p.CanProvide(tc.from, func(from label.Label) (*rule.Rule, bool) {
// 				return nil, false
// 			})

// 			if diff := cmp.Diff(tc.want, got); diff != "" {
// 				t.Errorf(".CanProvide (-want +got):\n%s", diff)
// 			}
// 		})
// 	}
// }
