package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Configure implements part of the language.Language interface
func (sl *scalaLang) Configure(c *config.Config, rel string, f *rule.File) {
	if f != nil {
		if err := getOrCreateScalaConfig(c, rel, sl).parseDirectives(f.Directives); err != nil {
			log.Fatalf("parsing directives in package %q: %v", rel, err)
		}
	}
}
