package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
)

// Configure implements part of the language.Language interface
func (sl *scalaLang) Configure(c *config.Config, rel string, f *rule.File) {
	sc := scalaconfig.GetOrCreate(sl.logger, sl, c, rel)
	if f != nil {
		if err := sc.ParseDirectives(f.Directives); err != nil {
			log.Fatalf("parsing directives in package %q: %v", rel, err)
		}
	}
}
