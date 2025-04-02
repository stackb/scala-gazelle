package scala

import (
	"os"
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

func TestScala(t *testing.T) {
	if val, ok := os.LookupEnv("SCALA_GAZELLE_DEBUG_PROCESS"); false && ok && (val == "1" || val == "true") {
		collections.PrintProcessIdForDelveAndWait()
	}
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"java_provider",
			"maven_provider",
			"override_provider",
			"protobuf_provider",
			"resolve_kind_rewrite_name",
			"source_provider",
			"scala_fileset",
		),
	).Run(t, "gazelle")
}
