package scalarule

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// ResolveContext carries context about a rule during rule provider import
// resolution.
type ResolveContext struct {
	Config    *config.Config
	RuleIndex *resolve.RuleIndex
	Rule      *rule.Rule
	From      label.Label
	File      *rule.File
}

// RuleProvider implementations are capable of providing a rule and import list
// to the gazelle GenerateArgs response.
type RuleProvider interface {
	// Kind of rule e.g. 'scala_library'
	Kind() string
	// Name of the rule.
	Name() string
	// Rule provides the gazelle rule implementation.
	Rule() *rule.Rule
	// Resolve performs deps resolution, similar to the gazelle Resolver
	// interface.
	Resolve(ctx *ResolveContext, importsRaw interface{})
	// Imports implements part of the resolve.Resolver interface.
	Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec
}
