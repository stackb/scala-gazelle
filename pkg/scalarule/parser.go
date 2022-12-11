package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Parser abstracts a service that can parse scala files.
type Parser interface {
	// ParseScala is used to parse a list of source files.  The list of srcs is
	// expected to be relative to the from.Pkg rel field, and the absolute path
	// of a file is expected at (dir, from.Pkg, src).  Kind is used to determine
	// if the rule is a test rule.
	ParseScala(from label.Label, kind string, dir string, srcs ...string) (*sppb.Rule, error)
}
