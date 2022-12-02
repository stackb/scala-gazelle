package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// ResolveImports implements the resolver.ImportResolver interface.
func (sl *scalaLang) ResolveImports(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imports ...*resolver.Import) {
	sl.importResolver.ResolveImports(c, ix, from, lang, imports...)
}
