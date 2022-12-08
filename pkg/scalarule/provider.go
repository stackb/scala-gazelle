package scalarule

import grule "github.com/bazelbuild/bazel-gazelle/rule"

// Provider is factory pattern capable of taking a config and package and
// returning a Provider.
type Provider interface {
	// Name returns the name of the rule
	Name() string
	// LoadInfo returns the gazelle LoadInfo.
	LoadInfo() grule.LoadInfo
	// KindInfo returns the gazelle KindInfo.
	KindInfo() grule.KindInfo
	// ProvideRule takes the given configuration and compilation and emits a
	// RuleProvider.  If the state of the ScalaConfiguration is such that the
	// rule should not be emitted, implementation should return nil.
	ProvideRule(rc *Config, pkg Package) RuleProvider
}
