package crossresolve

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

const scalaName = "scala"
const cmdGenerate = "generate"

func ExampleMavenCrossResolver_RegisterFlags_printdefaults() {
	os.Stderr = os.Stdout
	cr := NewMavenResolver(scalaName)
	got := flag.NewFlagSet(scalaName, flag.ExitOnError)
	c := &config.Config{}
	cr.RegisterFlags(got, cmdGenerate, c)
	got.PrintDefaults()
	// output:
	//	-maven_install_file string
	//     	pinned maven_install.json deps
	//   -maven_workspace_name string
	//     	name of the maven external workspace (default "maven")
}

func TestMavenCrossResolverRegisterFlags(t *testing.T) {
	for name, tc := range map[string]struct {
		args                   []string
		wantMavenInstallFile   string
		wantMavenWorkspaceName string
		files                  []testtools.FileSpec
	}{
		"typical usage": {
			args: []string{
				"-maven_install_file=./maven_install.json",
				"-maven_workspace_name=maven",
			},
			files: []testtools.FileSpec{
				{
					Path:    "maven_install.json",
					Content: "{}",
				},
			},
			wantMavenInstallFile:   "./maven_install.json",
			wantMavenWorkspaceName: "maven",
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
			if diff := cmp.Diff(tc.wantMavenInstallFile, cr.mavenInstallFile); diff != "" {
				t.Errorf(".mavenInstallFile (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantMavenWorkspaceName, cr.mavenWorkspaceName); diff != "" {
				t.Errorf(".mavenWorkspaceName (-want +got):\n%s", diff)
			}
		})
	}
}
