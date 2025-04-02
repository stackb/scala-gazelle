package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
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
	// Files returns the list of files in the Rule
	Files() []*sppb.File
}
