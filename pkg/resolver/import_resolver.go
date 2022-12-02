package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// ImportResolver knows how to resolve imports.
type ImportResolver interface {
	// Resolve takes the given config, gazelle rule index, and variadic list of
	// imports to resolve. The supplied imports are assigned known providers (or
	// error)
	ResolveImports(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imports ...*Import)
}
