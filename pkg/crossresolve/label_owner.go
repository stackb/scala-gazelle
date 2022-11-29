package crossresolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// LabelOwner is an optional interface for a cross-resolver
// that can claims a particular sub-space of labels.  For example, the
// maven resolver may return true for labels like "@maven//:junit_junit".
// the ruleIndex can be used to consult what type of label from is, based
// on the rule characteristics.  If no rule corresponding to the given
// label is found, ruleIndex returns nil, false.
type LabelOwner interface {
	IsLabelOwner(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool
}
