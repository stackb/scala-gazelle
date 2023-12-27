package scala

import (
	"log"
	"time"

	"github.com/bazelbuild/bazel-gazelle/label"
	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

const debugCache = false

func (sl *scalaLang) readScalaRuleCacheFile() error {
	t1 := time.Now()

	if err := protobuf.ReadFile(sl.cacheFileFlagValue, &sl.cache); err != nil {
		return err
	}

	if sl.cacheKeyFlagValue != sl.cache.Key {
		if debugCache {
			log.Printf("scala-gazelle cache invalidated! (want %q, got %q)", sl.cacheFileFlagValue, sl.cache.Key)
		}
		sl.cache = scpb.Cache{}
		return nil
	}

	parser.SortRules(sl.cache.Rules)

	for _, rule := range sl.cache.Rules {
		from, err := label.Parse(rule.Label)
		if err != nil {
			return err
		}
		if err := sl.parser.LoadScalaRule(from, rule); err != nil {
			return err
		}
	}

	if debugCache {
		t2 := time.Since(t1).Round(1 * time.Millisecond)
		log.Printf("Read cache %s (%d rules) %v", sl.cacheFileFlagValue, len(sl.cache.Rules), t2)
	}

	return nil
}

func (sl *scalaLang) writeScalaRuleCacheFile() error {
	sl.cache.PackageCount = int32(sl.packages.Size())
	sl.cache.Rules = sl.parser.ScalaRules()
	sl.cache.Key = sl.cacheKeyFlagValue

	if debugCache {
		log.Printf("Wrote scala-gazelle cache %s (%d rules)", sl.cacheFileFlagValue, len(sl.cache.Rules))
	}

	return protobuf.WriteFile(sl.cacheFileFlagValue, &sl.cache)
}
