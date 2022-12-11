package scala

import (
	"log"
	"sort"
	"time"

	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

const debugCache = true

// getScalaRules gets all the scala rules
func (p *scalaLang) getScalaRules() []*sppb.Rule {
	rules := make([]*sppb.Rule, 0, len(p.knownScalaRules))
	for _, rule := range p.knownScalaRules {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		a := rules[i]
		b := rules[j]
		return a.Label < b.Label
	})
	return rules
}

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
		if err := sl.sourceProvider.LoadScalaRule(from, rule); err != nil {
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
	sl.cache.Rules = sl.getScalaRules()

	if debugCache {
		log.Printf("Wrote cache %s (%d rules)", sl.cacheFileFlagValue, len(sl.cache.Rules))
	}

	return protobuf.WriteFile(sl.cacheFileFlagValue, sl.cache)
}
