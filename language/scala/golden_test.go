package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"maven_provider",
			"stackb_rules_proto_provider",
			"scalasource_provider",
			"override_provider",
			"jarindex_provider",
		),
	).Run(t, "gazelle")
}
