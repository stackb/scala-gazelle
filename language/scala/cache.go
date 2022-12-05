package scala

import (
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

func (sl *scalaLang) readCacheFile() error {
	if err := protobuf.ReadFile(sl.cacheFileFlagValue, sl.cache); err != nil {
		return err
	}
	for _, rule := range sl.cache.Rules {
		if err := sl.sourceProvider.ProvideRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func (sl *scalaLang) writeCacheFile() error {
	sl.cache.PackageCount = int32(len(sl.packages))
	sl.cache.Rules = sl.sourceProvider.ProvidedRules()
	return protobuf.WriteFile(sl.cacheFileFlagValue, sl.cache)
}
