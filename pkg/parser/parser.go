package parser

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Parser abstracts a service that can parse scala files.  It also supports a
// load operation to facilitate a cache.
type Parser interface {
	// LoadScalaRule loads the given rule state.
	LoadScalaRule(from label.Label, rule *sppb.Rule) error

	// ParseScalaRule is used to parse a list of source files.  The srcs list
	// is expected to be relative to dir.
	ParseScalaRule(kind string, from label.Label, dir string, srcs ...string) (*sppb.Rule, error)
}
