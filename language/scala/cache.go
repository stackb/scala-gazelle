package scala

import (
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
	sl.cache.PackageCount = int32(len(sl.packages))
	sl.cache.Rules = sl.sourceResolver.Rules()
	return protobuf.WriteFile(sl.cacheFile, sl.cache)
}