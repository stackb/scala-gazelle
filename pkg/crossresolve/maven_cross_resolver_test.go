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

func TestMavenCrossResolverIsLabelOwner(t *testing.T) {
	for name, tc := range map[string]struct {
		mavenInstallJsonContent string
		lang                    string
		from                    label.Label
		indexFunc               func(from label.Label) (*rule.Rule, bool)
		want                    bool
	}{
		"degenerate case": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    scalaName,
			from:                    label.NoLabel,
			want:                    false,
		},
		"managed maven dependency": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    scalaName,
			from:                    label.New("maven", "", "com_guava_guava"),
			want:                    true,
		},
		"unmanaged non-maven dependency": {
			mavenInstallJsonContent: mavenInstallJsonSimpleExample,
			lang:                    scalaName,
			from:                    label.New("not-maven", "", "com_guava_guava"),
			want:                    false,
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

			got := cr.IsLabelOwner(tc.from, tc.indexFunc)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".IsLabelOwner (-want +got):\n%s", diff)
			}
		})
	}
}
