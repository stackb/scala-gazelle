package scalarule

import "github.com/bazelbuild/bazel-gazelle/rule"

// RuleResolver is an optional interface for a RuleInfo implementation.  This is
// a mechanism for rule implementations to only modify an existing rule rather
// than having to create one from scratch.
type RuleResolver interface {
	// ResolveRule takes the given configuration emits a RuleProvider. If the
	// state of the package is such that the rule should not managed, return nil.
	ResolveRule(rc *Config, pkg Package, existing *rule.Rule) RuleProvider
}
