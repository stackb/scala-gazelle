package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			// "maven_direct_deps",
			"rules_jvm_external_provider",
			"stackb_rules_proto_provider",
			"source_scala_provider",
		),
	).Run(t, "gazelle")
}
