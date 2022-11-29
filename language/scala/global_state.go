package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// globalState is an interface that any scala config has access to
type globalState interface {
	// LookupRule is a function that returns the generated rule for the given label
	LookupRule(from label.Label) (*rule.Rule, bool)
}

// LookupRule implements part of the globalState interface
func (sl *scalaLang) LookupRule(from label.Label) (*rule.Rule, bool) {
	r, ok := sl.allRules[from]
	if ok {
		log.Printf("hit scalaLang.LookupRule(%v): %s/%s", from, r.Kind(), r.Name())
	} else {
		log.Printf("miss scalaLang.LookupRule(%v)", from)
	}
	return r, ok
}

// setGlobalRule records the given rule in the global map.
func (sl *scalaLang) setGlobalRule(from label.Label, r *rule.Rule) {
	log.Println("setGlobalRule", from, r.Kind(), r.Name())
	sl.allRules[from] = r
}
