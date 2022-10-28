package crossresolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// ScalaFileParser abstracts a service that can parse scala files.
type ScalaFileParser interface {
	// ParseScalaFiles is used to parse a list of source files.  The list of srcs
	// is expected to be relative to the from.Pkg rel field, and the absolute path
	// of a file is expected at (dir, from.Pkg, src).  Kind is used to determine
	// if the rule is a test rule.
	ParseScalaFiles(dir string, from label.Label, kind string, srcs ...string) (*index.ScalaRuleSpec, error)
}
