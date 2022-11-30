package crossresolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// ScalaRuleParser abstracts a service that can parse scala files.
type ScalaRuleParser interface {
	// ParseScalaRule is used to parse a list of source files.  The list of srcs
	// is expected to be relative to the from.Pkg rel field, and the absolute path
	// of a file is expected at (dir, from.Pkg, src).  Kind is used to determine
	// if the rule is a test rule.
	ParseScalaRule(dir string, from label.Label, kind string, srcs ...string) (*sppb.Rule, error)
}
