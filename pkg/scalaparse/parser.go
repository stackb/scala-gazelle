package scalaparse

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Parser abstracts a service that can parse scala files.
type Parser interface {
	// ParseScalaFiles is used to parse a list of source files.  The file list is
	// expected to be relative to dir.  The rule argument is modified by the
	// operation. A new copy of the parsed files is returned.  The list of files
	// that were actually parsed may be a subset of the rule argument.
	ParseScalaFiles(from label.Label, kind, dir string, srcs ...string) ([]*sppb.File, error)
}
