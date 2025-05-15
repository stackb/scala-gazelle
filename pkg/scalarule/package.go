package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/language"
	grule "github.com/bazelbuild/bazel-gazelle/rule"
)

// Package is responsible for instantiating a Rule interface for the given
// gazelle.Rule, parsing the attribute name given (typically 'srcs').
type Package interface {
	// NewScalaRule creates new scalarule.Rule from the given rule.Rule.
	NewScalaRule(r *grule.Rule) (Rule, error)
	// GenerateArgs returns the GenerateArgs for the package
	GenerateArgs() language.GenerateArgs
	// GeneratedRules returns a list of generated rules in the package.
	GeneratedRules() []*grule.Rule
}
