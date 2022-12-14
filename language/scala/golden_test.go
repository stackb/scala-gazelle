package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"maven_provider",
			// "protobuf_provider",
			// "source_provider",
			// "override_provider",
			// "java_provider",
		),
	).Run(t, "gazelle")
}
