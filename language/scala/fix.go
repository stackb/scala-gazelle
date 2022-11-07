package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix implements part of the language.Language interface
func (sl *scalaLang) Fix(c *config.Config, f *rule.File) {
}
