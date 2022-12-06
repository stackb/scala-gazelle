package scala

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
)

// onGenerate is called on the the first GenerateRules call.
func (sl *scalaLang) onGenerate() error {
	if err := sl.sourceProvider.Start(); err != nil {
		return fmt.Errorf("starting parser: %w", err)
	}
	return nil
}

// onResolve is called when gazelle transitions from the generate phase to the resolve phase
func (sl *scalaLang) onResolve() {
	for _, provider := range sl.knownImportProviders {
		provider.OnResolve()
	}

	if sl.cacheFileFlagValue != "" {
		if err := sl.writeCacheFile(); err != nil {
			log.Fatalf("failed to write cache: %v", err)
		}
	}
}

// onEnd is called when the last rule has been resolved.
func (sl *scalaLang) onEnd() {
	if sl.cpuprofileFlagValue != "" {
		pprof.StopCPUProfile()
	}

	if sl.memprofileFlagValue != "" {
		f, err := os.Create(sl.memprofileFlagValue)
		if err != nil {
			log.Fatalf("creating memprofile: %v", err)
		}
		log.Println("Writing memprofile to", sl.memprofileFlagValue)
		pprof.WriteHeapProfile(f)
	}

}
