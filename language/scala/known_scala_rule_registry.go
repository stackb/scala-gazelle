package scala

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// GetKnownScalaRule implements part of the resolver.KnownScalaRuleRegistry
// interface.
func (sl *scalaLang) GetKnownScalaRule(from label.Label) (*sppb.Rule, bool) {
	r, ok := sl.knownScalaRules[from]
	return r, ok
}

// PutKnownScalaRule implements part of the resolver.KnownScalaRuleRegistry
// interface.
func (sl *scalaLang) PutKnownScalaRule(from label.Label, r *sppb.Rule) error {
	if _, ok := sl.knownRules[from]; ok {
		return fmt.Errorf("duplicate known rule: %s", from)
	}
	sl.knownScalaRules[from] = r
	return nil
}
