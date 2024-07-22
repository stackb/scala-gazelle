package scalaconfig

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func NewTestScalaConfig(t *testing.T, universe resolver.Universe, rel string, dd ...rule.Directive) (*Config, error) {
	c := config.New()
	sc := New(universe, c, rel)
	err := sc.ParseDirectives(dd)
	return sc, err
}
