package scala

import (
	"log"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// onResolve is called when gazelle transitions from the generate phase to the
// resolve phase
func (sl *scalaLang) onResolve() {
	sl.phaseTransition("resolve")

	for _, provider := range sl.symbolProviders {
		if err := provider.OnResolve(); err != nil {
			log.Fatalf("provider.OnResolve transition error %s: %v", provider.Name(), err)
		}
	}

	// assign final readonly scala-specific scope
	if scalaScope, err := resolver.NewScalaScope(sl.globalScope); err != nil {
		sl.logger.Printf("warning: setting up global resolver scope: %v", err)
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
	sl.phaseTransition("end")

	for _, provider := range sl.symbolProviders {
		if err := provider.OnEnd(); err != nil {
			log.Fatalf("provider.OnEnd transition error %s: %v", provider.Name(), err)
		}
	}

	sl.dumpResolvedImportMap()
	sl.reportCoverage(log.Printf)
	sl.stopCpuProfiling()
	sl.stopMemoryProfiling()

	if sl.logFile != nil {
		sl.logFile.Close()
	}
}

func (sl *scalaLang) phaseTransition(phase string) {
	sl.logger.Debug().Msgf("transitioning to phase: %s", phase)
}
