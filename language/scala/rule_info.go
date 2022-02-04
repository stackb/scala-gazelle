package scala

import "github.com/bazelbuild/bazel-gazelle/rule"

// RuleInfo is capable of taking a compilation and deriving another rule
// based on it.
type RuleInfo interface {
	// Name returns the name of the rule
	Name() string
	// LoadInfo returns the gazelle LoadInfo.
	LoadInfo() rule.LoadInfo
	// KindInfo returns the gazelle KindInfo.
	KindInfo() rule.KindInfo
	// ProvideRule takes the given configration and compilation and emits a
	// RuleProvider.  If the state of the ScalaConfiguration is such that the
	// rule should not be emitted, implementation should return nil.
	ProvideRule(rc *RuleConfig) RuleProvider
}
