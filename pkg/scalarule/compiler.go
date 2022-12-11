package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Compiler abstracts a service that can compile scala files for their symbols.
type Compiler interface {
	// CompileScala is used to compile a list of source files.  The list of srcs
	// is expected to be relative to the from.Pkg rel field, and the absolute
	// path of a file is expected at (dir, from.Pkg, src).
	CompileScala(from label.Label, kind string, dir string, srcs ...string) (*sppb.Rule, error)
}
