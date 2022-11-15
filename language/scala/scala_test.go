package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			// "maven_resolver",
			"maven_direct_deps",
			// "proto_resolver",
		),
	).Run(t, "gazelle")
}

// TODO: re-enable whatever this was
// func TestScala(t *testing.T) {
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	goldentest.FromDir("language/scala",
// 		goldentest.WithOnlyTests("greeter"),
// 		goldentest.WithDataFiles(bazel.RunfileEntry{
// 			Path:      filepath.Join(cwd, "sourceindexer"),
// 			ShortPath: "sourceindexer",
// 		}, bazel.RunfileEntry{
// 			Path:      filepath.Join(cwd, "scala_compiler.jar"),
// 			ShortPath: "scala_compiler.jar",
// 		})).
// 		Run(t, "gazelle")
// }
