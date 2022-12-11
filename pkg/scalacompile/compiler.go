package scalacompile

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Compiler abstracts a service that can compile a list of files and return the set of
// symbols named in the files.
type Compiler interface {
	// CompileScalaRule is used to compile a list of source files.  The file list is
	// expected to be relative to dir.  The rule argument is modified by the
	// operation such that the file symbols are overwritten.
	CompileScalaRule(from label.Label, dir string, rule *sppb.Rule) error
}
