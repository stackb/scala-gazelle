package scala

import (
	"fmt"
	"log"
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

	sl.scalaCompiler.OnResolve()

	if sl.cacheFileFlagValue != "" {
		if err := sl.writeCacheFile(); err != nil {
			log.Fatalf("failed to write cache: %v", err)
		}
	}
}

// onEnd is called when the last rule has been resolved.
func (sl *scalaLang) onEnd() {
	sl.scalaCompiler.stop()
}
