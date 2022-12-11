package scalaparse

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

type Reader interface {
	// ReadScalaRule loads the given rule state.
	ReadScalaRule(from label.Label, rule *sppb.Rule) error
	// ScalaRules returns all the stores rules.
	ScalaRules() []*sppb.Rule
}
