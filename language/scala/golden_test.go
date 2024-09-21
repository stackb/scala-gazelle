package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"java_provider",
			// "maven_provider",
			// "override_provider",
			// "protobuf_provider",
			// "resolve_kind_rewrite_name",
			// "source_provider",
		),
	).Run(t, "gazelle")
}
