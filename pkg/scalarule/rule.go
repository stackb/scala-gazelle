package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// Rule represents a collection of files with their imports and exports.
type Rule interface {
	// Exports returns the list of provided symbols that are importable by other
	// rules.
	Provides() []resolve.ImportSpec
	// Import returns the list of required imports for the rule.
	Imports(from label.Label) resolver.ImportMap
	// Import returns the list of required exports for the rule.
	Exports(from label.Label) resolver.ImportMap
}
