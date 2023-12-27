package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Kinds implements part of the language.Language interface
func (sl *scalaLang) Kinds() map[string]rule.KindInfo {
	kinds := make(map[string]rule.KindInfo)

	for _, name := range sl.ruleProviderRegistry.ProviderNames() {
		if provider, ok := sl.ruleProviderRegistry.LookupProvider(name); ok {
			kinds[provider.Name()] = provider.KindInfo()
		} else {
			log.Fatal("rule provider not found:", name)
		}
	}

	return kinds
}
