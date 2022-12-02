package scala

import (
	"fmt"
	"log"

	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

// onGenerate is called on the the first GenerateRules call.
func (sl *scalaLang) onGenerate() error {
	if err := sl.sourceResolver.Start(); err != nil {
		return fmt.Errorf("starting parser: %w", err)
	}
	return nil
}

// onResolve is called when gazelle transitions from the generate phase to the resolve phase
func (sl *scalaLang) onResolve() {

	for _, r := range sl.resolvers {
		if l, ok := r.(crossresolve.GazellePhaseTransitionListener); ok {
			l.OnResolve()
		}
	}

	sl.scalaCompiler.OnResolve()

	if sl.cacheFile != "" {
		if err := sl.writeCacheFile(); err != nil {
			log.Fatalf("failed to write cache: %v", err)
		}
	}
}

// onEnd is called when the last rule has been resolved.
func (sl *scalaLang) onEnd() {
	sl.scalaCompiler.stop()
}
