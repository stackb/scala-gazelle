package scala

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	goldentest.FromDir("language/scala", goldentest.WithDataFiles(bazel.RunfileEntry{
		Path:      filepath.Join(cwd, "sourceindexer"),
		ShortPath: "sourceindexer",
	})).Run(t, "gazelle")
}
