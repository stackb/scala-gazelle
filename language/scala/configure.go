package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KnownDirectives implements part of the language.Language interface
func (*scalaLang) KnownDirectives() []string {
	return []string{
		ruleDirective,
		overrideDirective,
		indirectDependencyDirective,
		scalaExplainDependencies,
		mapKindImportNameDirective,
	}
}

// Configure implements part of the language.Language interface
func (sl *scalaLang) Configure(c *config.Config, rel string, f *rule.File) {
	if f == nil {
		return
	}
	if err := getOrCreateScalaConfig(c).parseDirectives(rel, f.Directives); err != nil {
		log.Fatalf("error while parsing rule directives in package %q: %v", rel, err)
	}
}
