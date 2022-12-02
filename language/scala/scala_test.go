package scala

import (
	"testing"

	"github.com/stackb/rules_proto/pkg/goldentest"
)

func TestScala(t *testing.T) {
	goldentest.FromDir("language/scala",
		goldentest.WithOnlyTests(
			"maven_resolver",
			"maven_direct_deps",
			"proto_resolver",
		),
	).Run(t, "gazelle")
}
