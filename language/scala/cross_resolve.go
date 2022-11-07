package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// CrossResolve implements part of the resolve.CrossResolver interface
func (sl *scalaLang) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	for _, r := range sl.resolvers {
		if result := r.CrossResolve(c, ix, imp, lang); len(result) > 0 {
			// log.Printf("scala.CrossResolve hit %T %s", r, imp.Imp)
			return result
		}
	}
	if result := sl.importRegistry.CrossResolve(c, ix, imp, lang); len(result) > 0 {
		// log.Printf("scala.CrossResolve hit %T %s", sl.importRegistry, imp.Imp)
		return result
	}
	return nil
}