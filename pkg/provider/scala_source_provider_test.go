package provider

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestScalaSourceProvider(t *testing.T) {
	for name, tc := range map[string]struct {
		args      []string
		kind      string
		from      label.Label
		testfiles []string // name(s) of files under testdata/
		want      []*resolver.KnownImport
	}{
		"GreeterClient.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/GreeterClient.scala"},
			want: []*resolver.KnownImport{
				{
					Type:   sppb.ImportType_OBJECT,
					Import: "examples.helloworld.greeter.GreeterClient",
					Label:  label.Label{Name: "greeter_lib"},
				},
				{
					Type:   sppb.ImportType_PACKAGE,
					Import: "examples.helloworld.greeter",
					Label:  label.Label{Name: "greeter_lib"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			testutil.ListFiles(t, dir)

			p := NewScalaSourceProvider()

			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: dir,
			}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			importRegistry := &mockKnownImportRegistry{}

			if err := p.CheckFlags(fs, c, importRegistry); err != nil {
				t.Fatal(err)
			}

			if err := p.Start(); err != nil {
				t.Fatal(err)
			}

			r, err := p.ParseScalaFiles(tc.from, tc.kind, dir, tc.testfiles...)
			p.OnResolve()
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range r.Files {
				if file.Error != "" {
					t.Fatal("parse rule file error:", file.Error)
				}
			}

			if diff := cmp.Diff(tc.want, importRegistry.got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
