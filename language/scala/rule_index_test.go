package scala

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

type mockRuleIndex struct {
	// from records the argument given to LookupRule
	from label.Label
}

// LookupRule implements part of the crossresolve.RuleIndex interface
func (m *mockRuleIndex) LookupRule(from label.Label) (*rule.Rule, bool) {
	m.from = from
	return nil, false
}
