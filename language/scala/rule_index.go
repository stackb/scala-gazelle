package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const debugLookupRule = false

// LookupRule implements part of the crossresolve.RuleIndex interface
func (sl *scalaLang) LookupRule(from label.Label) (*rule.Rule, bool) {
	r, ok := sl.allRules[from]
	if debugLookupRule {
		log.Printf("scalaLang.LookupRule(%q) -> %t", from, ok)
	}
	return r, ok
}

// recordRule sets the given rule in the global label->rule map.
func (sl *scalaLang) recordRule(from label.Label, r *rule.Rule) {
	if debugLookupRule {
		log.Printf("scalaLang.recordRule(%q) [%s]", from, r.Kind())
	}
	sl.allRules[from] = r
}
