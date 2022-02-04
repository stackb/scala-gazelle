package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	scalaLibraryRuleName   = "scala_library"
	scalaLibraryRuleSuffix = "_scala_library"
)

func init() {
	Rules().MustRegisterRule("stackb:rules_scala:"+scalaLibraryRuleName,
		&scalaLibrary{
			kindName: scalaLibraryRuleName,
		})
}

// scalaLibrary implements RuleInfo for the 'scala_library' rule from
// @rules_scala.
type scalaLibrary struct {
	kindName string
}

// Name implements part of the RuleInfo interface.
func (s *scalaLibrary) Name() string {
	return s.kindName
}

// KindInfo implements part of the RuleInfo interface.
func (s *scalaLibrary) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		MergeableAttrs: map[string]bool{
			"srcs":    true,
			"exports": true,
		},
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the RuleInfo interface.
func (s *scalaLibrary) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    "@io_bazel_rules_scala//scala:scala.bzl",
		Symbols: []string{s.kindName},
	}
}

// ProvideRule implements part of the RuleInfo interface.
func (s *scalaLibrary) ProvideRule(cfg *RuleConfig) RuleProvider {
	return &scalaLibraryRule{
		kindName:       s.kindName,
		ruleNameSuffix: scalaLibraryRuleSuffix,
	}
}

// scalaLibraryRule implements RuleProvider for 'scala_library'-derived rules.
type scalaLibraryRule struct {
	kindName       string
	ruleNameSuffix string
	srcs           []string
	ruleConfig     *RuleConfig
}

// Kind implements part of the ruleProvider interface.
func (s *scalaLibraryRule) Kind() string {
	return s.kindName
}

// Name implements part of the ruleProvider interface.
func (s *scalaLibraryRule) Name() string {
	return "fixme" + s.ruleNameSuffix
}

// Srcs computes the srcs list for the rule.
func (s *scalaLibraryRule) Srcs() []string {
	srcs := make([]string, 0)
	return srcs
}

// Deps computes the deps list for the rule.
func (s *scalaLibraryRule) Deps() []string {
	deps := s.ruleConfig.GetDeps()
	return DeduplicateAndSort(deps)
}

// Rule implements part of the ruleProvider interface.
func (s *scalaLibraryRule) Rule() *rule.Rule {
	newRule := rule.NewRule(s.Kind(), s.Name())

	newRule.SetAttr("srcs", s.Srcs())

	deps := s.Deps()
	if len(deps) > 0 {
		newRule.SetAttr("deps", deps)
	}

	// // set the override language such that deps of 'proto_scala_library' and
	// // 'grpc_scala_library' can resolve together (matches the value used by
	// // "Imports").
	// newRule.SetPrivateAttr(protoc.ResolverImpLangPrivateKey, scalaLibraryRuleSuffix)

	return newRule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaLibraryRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	// if lib, ok := r.PrivateAttr(protoc.ProtoLibraryKey).(protoc.ProtoLibrary); ok {
	// 	return protoc.ProtoLibraryImportSpecsForKind(scalaLibraryRuleSuffix, lib)
	// }
	return nil
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaLibraryRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
	// if s.resolver == nil {
	// 	return
	// }
	// s.resolver(c, ix, r, imports, from)
}
