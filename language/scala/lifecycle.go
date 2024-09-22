package scala

import (
	"log"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// onResolve is called when gazelle transitions from the generate phase to the
// resolve phase
func (sl *scalaLang) onResolve() {
	for _, provider := range sl.symbolProviders {
		if err := provider.OnResolve(); err != nil {
			log.Fatalf("provider.OnResolve transition error %s: %v", provider.Name(), err)
		}
	}

	// assign final readonly scala-specific scope
	if scalaScope, err := resolver.NewScalaScope(sl.globalScope); err != nil {
		log.Printf("warning: error while setting up final scope: %v (some builtin symbols may not resolve!)", err)
	} else {
		sl.globalScope = scalaScope
	}

	if sl.cacheFileFlagValue != "" {
		if err := sl.writeScalaRuleCacheFile(); err != nil {
			log.Fatalf("failed to write cache: %v", err)
		}
	}
}

// onEnd is called when the last rule has been resolved.
func (sl *scalaLang) onEnd() {
	for _, provider := range sl.symbolProviders {
		if err := provider.OnEnd(); err != nil {
			log.Fatalf("provider.OnEnd transition error %s: %v", provider.Name(), err)
		}
	}

	sl.dumpResolvedImportMap()
	sl.reportCoverage(log.Printf)
	sl.stopCpuProfiling()
	sl.stopMemoryProfiling()
}
