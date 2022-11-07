package crossresolve

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

const scalaName = "scala"
const cmdGenerate = "generate"
const mavenInstallJsonSimpleExample = `{
	"dependency_tree": {
		"dependencies": [
			{
				"coord": "xml-apis:xml-apis:1.4.01",
				"dependencies": [],
				"directDependencies": [],
				"exclusions": [
					"log4j:log4j"
				],
				"file": "v1/https/repo.maven.apache.org/maven2/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar",
				"mirror_urls": [
					"https://repo.maven.apache.org/maven2/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar",
					"https://omnistac.jfrog.io/artifactory/libs-release/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar"
				],
				"packages": [
					"javax.xml",
					"javax.xml.datatype",
					"javax.xml.namespace",
					"javax.xml.parsers",
					"javax.xml.stream",
					"javax.xml.stream.events",
					"javax.xml.stream.util",
					"javax.xml.transform",
					"javax.xml.transform.dom",
					"javax.xml.transform.sax",
					"javax.xml.transform.stax",
					"javax.xml.transform.stream",
					"javax.xml.validation",
					"javax.xml.xpath",
					"org.apache.xmlcommons",
					"org.w3c.dom",
					"org.w3c.dom.bootstrap",
					"org.w3c.dom.css",
					"org.w3c.dom.events",
					"org.w3c.dom.html",
					"org.w3c.dom.ls",
					"org.w3c.dom.ranges",
					"org.w3c.dom.stylesheets",
					"org.w3c.dom.traversal",
					"org.w3c.dom.views",
					"org.w3c.dom.xpath",
					"org.xml.sax",
					"org.xml.sax.ext",
					"org.xml.sax.helpers"
				],
				"sha256": "a840968176645684bb01aed376e067ab39614885f9eee44abe35a5f20ebe7fad",
				"url": "https://repo.maven.apache.org/maven2/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar"
			}
		],
		"version": "0.1.0"
	}
}`

func ExampleMavenCrossResolver_RegisterFlags_printdefaults() {
	os.Stderr = os.Stdout
	cr := NewMavenResolver(scalaName)
	got := flag.NewFlagSet(scalaName, flag.ExitOnError)
	c := &config.Config{}
	cr.RegisterFlags(got, cmdGenerate, c)
	got.PrintDefaults()
	// output:
	//	-pinned_maven_install_json_files string
	//     	comma-separated list of maven_install pinned deps files
}

func TestMavenCrossResolverFlags(t *testing.T) {
	for name, tc := range map[string]struct {
		args                   []string
		wantMavenInstall       string
		wantMavenWorkspaceName string
		files                  []testtools.FileSpec
	}{
		"typical usage": {
			args: []string{
				"-pinned_maven_install_json_files=./maven_install.json",
			},
			files: []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: "{}",
				},
			},
			wantMavenInstall: "./maven_install.json",
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustPrepareTestFiles(t, tc.files)
			defer cleanup()

			cr := NewMavenResolver(scalaName)
			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
			}
			cr.RegisterFlags(fs, cmdGenerate, c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if err := cr.CheckFlags(fs, c); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.wantMavenInstall, cr.pinnedMavenInstallFlagValue); diff != "" {
				t.Errorf(".mavenInstallFile (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMavenCrossResolverCrossResolve(t *testing.T) {
	for name, tc := range map[string]struct {
		mavenInstallJsonContent string
		lang                    string
		imp                     resolve.ImportSpec
		want                    []resolve.FindResult
	}{
		"match (lang == r.lang)": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    scalaName,
			imp:                     resolve.ImportSpec{Lang: scalaName, Imp: "javax.xml.parsers"},
			want:                    []resolve.FindResult{{Label: label.New("maven", "", "xml_apis_xml_apis")}},
		},
		"match (lang matches even though imp.Lang does not)": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    scalaName,
			imp:                     resolve.ImportSpec{Lang: "scala3", Imp: "javax.xml.parsers"},
			want:                    []resolve.FindResult{{Label: label.New("maven", "", "xml_apis_xml_apis")}},
		},
		"match (r.lang matches imp.Lang)": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    "scala3",
			imp:                     resolve.ImportSpec{Lang: "scala3", Imp: "javax.xml.parsers"},
			want:                    []resolve.FindResult{{Label: label.New("maven", "", "xml_apis_xml_apis")}},
		},
		"no match (r.lang does not match lang or imp.Lang)": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    "scala2",
			imp:                     resolve.ImportSpec{Lang: "scala3", Imp: "javax.xml.parsers"},
			want:                    nil,
		},
		"no match (package unknown)": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    scalaName,
			imp:                     resolve.ImportSpec{Lang: scalaName, Imp: "com.foo.bar"},
			want:                    nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			files := []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: tc.mavenInstallJsonContent,
				},
			}

			tmpDir, _, cleanup := testutil.MustPrepareTestFiles(t, files)
			defer cleanup()

			args := []string{
				"-pinned_maven_install_json_files=./maven_install.json",
			}

			cr := NewMavenResolver(tc.lang)
			fs := flag.NewFlagSet(tc.lang, flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
			}
			cr.RegisterFlags(fs, cmdGenerate, c)
			if err := fs.Parse(args); err != nil {
				t.Fatal(err)
			}
			if err := cr.CheckFlags(fs, c); err != nil {
				t.Fatal(err)
			}

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)
			got := cr.CrossResolve(c, ix, tc.imp, scalaName)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".CrossResolve (-want +got):\n%s", diff)
			}
		})
	}
}
