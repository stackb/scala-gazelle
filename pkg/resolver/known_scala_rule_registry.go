package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// KnownScalaRuleRegistry is an index of known rules keyed by their label.
type KnownScalaRuleRegistry interface {
	// GetKnownRule does a lookup of the given label and returns the
	// known rule.  If not known `(nil, false)` is returned.
	GetKnownScalaRule(from label.Label) (*sppb.Rule, bool)

	// PutKnownRule adds the given known rule to the registry.
	PutKnownScalaRule(from label.Label, r *sppb.Rule) error
}
