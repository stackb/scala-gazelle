package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"maven_direct_deps",
			"maven_resolver",
			"proto_resolver",
			"source_resolver",
		),
	).Run(t, "gazelle")
}
