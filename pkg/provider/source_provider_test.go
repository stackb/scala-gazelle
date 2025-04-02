package provider_test

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/rs/zerolog"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

var update = flag.Bool("update", false, "update golden files")

func TestScalaSourceProviderParseScalaRule(t *testing.T) {
	rel := "pkg/provider"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
		dir = filepath.Join(bwd, rel)
	}
	t.Log("dir:", dir)

	scope := resolver.NewTrieScope()

	provider := provider.NewSourceProvider(zerolog.New(os.Stderr), func(msg string) {})

	fs := flag.NewFlagSet("", flag.ExitOnError)
	c := &config.Config{
		WorkDir: dir,
	}
	provider.RegisterFlags(fs, "update", c)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	if err := provider.CheckFlags(fs, c, scope); err != nil {
		t.Fatal(err)
	}
	defer provider.OnResolve()

	srcs, err := collections.CollectFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("srcs:", srcs)

	for _, src := range srcs {
		if filepath.Ext(src) != ".scala" {
			continue
		}
		t.Run(src, func(t *testing.T) {
			goldenFile := filepath.Join(dir, src+".golden.json")
			from := label.Label{Pkg: rel, Name: src}
			got, err := provider.ParseScalaRule("scala_library", from, dir, src)
			if err != nil {
				t.Fatal(err)
			}
			got.ParseTimeMillis = 0

			if *update {
				if err := protobuf.WritePrettyJSONFile(goldenFile, got); err != nil {
					t.Fatal(err)
				}
				log.Println("Wrote golden file:", goldenFile)
				return
			}

			var want sppb.Rule
			if err := protobuf.ReadFile(goldenFile, &want); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(&want, got,
				cmpopts.IgnoreUnexported(
					sppb.Rule{},
					sppb.File{},
					sppb.Symbol{},
					sppb.ClassList{}),
			); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}

		})
	}
}
