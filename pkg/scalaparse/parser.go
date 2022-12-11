package scalaparse

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Parser abstracts a service that can parse scala files.
type Parser interface {
	// ParseScalaFiles is used to parse a list of source files.  The srcs list
	// is expected to be relative to dir.
	ParseScalaFiles(from label.Label, kind, dir string, srcs ...string) ([]*sppb.File, error)
}
