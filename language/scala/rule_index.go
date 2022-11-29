package scala

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// LookupRule implements part of the crossresolve.RuleIndex interface
func (sl *scalaLang) LookupRule(from label.Label) (*rule.Rule, bool) {
	r, ok := sl.allRules[from]
	return r, ok
}

// recordRule sets the given rule in the global label->rule map.
func (sl *scalaLang) recordRule(from label.Label, r *rule.Rule) {
	sl.allRules[from] = r
}
