package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Kinds implements part of the language.Language interface
func (sl *scalaLang) Kinds() map[string]rule.KindInfo {
	kinds := make(map[string]rule.KindInfo)

	for _, name := range sl.ruleRegistry.RuleNames() {
		rule, err := sl.ruleRegistry.LookupRule(name)
		if err != nil {
			log.Fatal("Kinds:", err)
		}
		kinds[rule.Name()] = rule.KindInfo()
	}

	return kinds
}
