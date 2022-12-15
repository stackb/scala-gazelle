package scala

import (
	"log"
	"time"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

const debugCache = true

func (sl *scalaLang) readScalaRuleCacheFile() error {
	t1 := time.Now()

	if err := protobuf.ReadFile(sl.cacheFileFlagValue, sl.cache); err != nil {
		return err
	}
	for _, rule := range sl.cache.Rules {
		from, err := label.Parse(rule.Label)
		if err != nil {
			return err
		}
		if err := sl.parser.LoadScalaRule(from, rule); err != nil {
			return err
		}
	}

	t2 := time.Since(t1).Round(1 * time.Millisecond)

	if debugCache {
		log.Printf("Read cache %s (%d rules) %v", sl.cacheFileFlagValue, len(sl.cache.Rules), t2)
	}
	return nil
}

func (sl *scalaLang) writeScalaRuleCacheFile() error {
	sl.cache.PackageCount = int32(len(sl.packages))
	sl.cache.Rules = sl.parser.ScalaRules()

	if debugCache {
		log.Printf("Wrote cache %s (%d rules)", sl.cacheFileFlagValue, len(sl.cache.Rules))
	}

	return protobuf.WriteFile(sl.cacheFileFlagValue, sl.cache)
}
