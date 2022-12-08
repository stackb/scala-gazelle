package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// Rule represents a collection of files with their imports and exports.
type Rule interface {
	// Files is the list of files in the rule.
	Files() []*sppb.File
	// Exports returns the list of provided symbols that are importable by other
	// rules.
	Exports() []resolve.ImportSpec
	// Import returns the list of required imports for the rule.
	Imports() resolver.ImportMap
}
