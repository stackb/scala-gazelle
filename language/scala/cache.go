package scala

import (
	"sort"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

func (sl *scalaLang) readCacheFile() error {
	if err := protobuf.ReadFile(sl.cacheFile, sl.cache); err != nil {
		return err
	}
	for _, rule := range sl.cache.Rules {
		if err := sl.sourceResolver.AddRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func (sl *scalaLang) writeCacheFile() error {
	// record package count
	sl.cache.PackageCount = int32(len(sl.packages))

	// record rules - sorted by label
	ruleMap := sl.sourceResolver.Rules()
	rules := make([]*sppb.Rule, 0, len(ruleMap))
	for _, r := range ruleMap {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool {
		a := rules[i]
		b := rules[j]
		return a.Label < b.Label
	})
	sl.cache.Rules = rules

	return protobuf.WriteFile(sl.cacheFile, sl.cache)
}
