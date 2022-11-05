package crossresolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// ScalaSourceRuleRegistry keep track of which files are associated under a rule
// (which has a srcs attribute).
type ScalaSourceRuleRegistry interface {
	// GetScalaFiles returns the rule spec for a given label.  If the label is
	// unknown, false is returned.
	GetScalaRule(from label.Label) (*index.ScalaRuleSpec, bool)
	// GetScalaRules
	GetScalaRules() map[label.Label]*index.ScalaRuleSpec
	// GetScalaFile
	GetScalaFile(filename string) *index.ScalaFileSpec
}
