package scala

import "github.com/bazelbuild/bazel-gazelle/rule"

// RuleInfo is factory pattern capable of taking a config and package and
// returning a RuleProvider.
type RuleInfo interface {
	// Name returns the name of the rule
	Name() string
	// LoadInfo returns the gazelle LoadInfo.
	LoadInfo() rule.LoadInfo
	// KindInfo returns the gazelle KindInfo.
	KindInfo() rule.KindInfo
	// ProvideRule takes the given configuration and compilation and emits a
	// RuleProvider.  If the state of the ScalaConfiguration is such that the
	// rule should not be emitted, implementation should return nil.
	ProvideRule(rc *RuleConfig, pkg ScalaPackage) RuleProvider
}

// RuleResolver is an optional interface for a RuleInfo
// implementation.  This is a mechanism for rule implementations to only modify
// an existing rule rather than having to create one from scratch.
type RuleResolver interface {
	// ResolveRule takes the given configuration emits a RuleProvider.  If the
	// state of the package is such that the rule should not be
	// emitted, implementation should return nil.
	ResolveRule(rc *RuleConfig, pkg ScalaPackage, existing *rule.Rule) RuleProvider
}
