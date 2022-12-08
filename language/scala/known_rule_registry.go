package scala

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GetKnownRule implements part of the
// resolver.KnownRuleRegistry interface.
func (sl *scalaLang) GetKnownRule(from label.Label) (*rule.Rule, bool) {
	r, ok := sl.knownRules[from]
	return r, ok
}

// PutKnownRule implements part of the
// resolver.KnownRuleRegistry interface.
func (sl *scalaLang) PutKnownRule(from label.Label, r *rule.Rule) error {
	if _, ok := sl.knownRules[from]; ok {
		return fmt.Errorf("duplicate known rule: %s", from)
	}
	sl.knownRules[from] = r
	return nil
}
