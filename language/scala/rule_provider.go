package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// RuleProvider implementations are capable of providing a rule and import list
// to the gazelle GenerateArgs response.
type RuleProvider interface {
	// Kind of rule e.g. 'scala_library'
	Kind() string
	// Name provides the name of the rule.
	Name() string
	// Rule provides the gazelle rule implementation.  A list of other generating rules in
	// the package are provided.
	Rule() *rule.Rule
	// Resolve performs deps resolution, similar to the gazelle Resolver
	// interface.  Imports here are always the scala_library file .proto
	// imports.
	Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label)
	// Imports implements part of the Resolver interface
	Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec
}
