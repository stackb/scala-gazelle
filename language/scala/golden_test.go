package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"rules_jvm_external_provider",
			"stackb_rules_proto_provider",
			"scalaparse_provider",
			"override_provider",
		),
	).Run(t, "gazelle")
}
