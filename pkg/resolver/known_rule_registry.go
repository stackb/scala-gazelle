package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KnownRuleRegistry is an index of known rules keyed by their label.
type KnownRuleRegistry interface {
	// GetKnownRule does a lookup of the given label and returns the
	// known rule.  If not known `(nil, false)` is returned.
	GetKnownRule(from label.Label) (*rule.Rule, bool)

	// PutKnownRule adds the given known rule to the registry.  It is an
	// error to attempt duplicate registration of the same rule twice.
	// Implementations should use the google.golang.org/grpc/status.Errorf for
	// error types.
	PutKnownRule(from label.Label, r *rule.Rule) error
}
