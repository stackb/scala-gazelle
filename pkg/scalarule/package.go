package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/language"
	grule "github.com/bazelbuild/bazel-gazelle/rule"
)

// Package is responsible for instantiating a Rule interface for the given
// gazelle.Rule, parsing the attribute name given (typically 'srcs').
type Package interface {
	// ParseRule parses the sources from the named attr (typically 'srcs') and
	// created a new Rule.
	ParseRule(r *grule.Rule, attrName string) (Rule, error)
	// GenerateArgs returns the GenerateArgs for the package
	GenerateArgs() language.GenerateArgs
	// GeneratedRules returns a list of generated rules in the package.
	GeneratedRules() []*grule.Rule
}
