package crossresolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// RuleIndex is an index of known rules indexed by their label.
type RuleIndex interface {
	// LookupRule is a function that returns the generated rule for the given label
	LookupRule(from label.Label) (*rule.Rule, bool)
}
