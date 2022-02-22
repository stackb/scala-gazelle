package scala

import (
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	// scalaLibraryRuleName is the name of the rule
	scalaLibraryRuleName = "scala_library"
	// scalaImportsPrivateKey is the PrivateAttr() key that holds the set of
	// imports for the rule.
	scalaImportsPrivateKey = "_scala_imports"
	// scalaLibraryRuleSuffix is the default suffix for generated rule names.
	scalaLibraryRuleSuffix = "_scala_library"
)

func init() {
	Rules().MustRegisterRule("stackb:rules_scala:experimental:"+scalaLibraryRuleName,
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
func (s *scalaLibrary) ProvideRule(cfg *RuleConfig, pkg ScalaPackage) RuleProvider {
	// files := pkg.Files()
	// if len(files) == 0 {
	// 	return nil
	// }

	return &scalaLibraryRule{
		kindName:       s.kindName,
		ruleNameSuffix: scalaLibraryRuleSuffix,
		ruleConfig:     cfg,
		rel:            pkg.Rel(),
		// files:          files,
	}
}

// scalaLibraryRule implements RuleProvider for 'scala_library'-derived rules.
type scalaLibraryRule struct {
	rel            string
	kindName       string
	ruleNameSuffix string
	ruleConfig     *RuleConfig
}

// Kind implements part of the ruleProvider interface.
func (s *scalaLibraryRule) Kind() string {
	return s.kindName
}

// Name implements part of the ruleProvider interface.
func (s *scalaLibraryRule) Name() string {
	prefix := filepath.Base(s.rel)
	return prefix + s.ruleNameSuffix
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

// imports computes the set of (scala) imports for the rule.
func (s *scalaLibraryRule) imports() []string {
	imps := make([]string, 0)
	return DeduplicateAndSort(imps)
}

// Rule implements part of the ruleProvider interface.
func (s *scalaLibraryRule) Rule() *rule.Rule {
	newRule := rule.NewRule(s.Kind(), s.Name())

	newRule.SetAttr("srcs", s.Srcs())

	deps := s.Deps()
	if len(deps) > 0 {
		newRule.SetAttr("deps", deps)
	}

	newRule.SetPrivateAttr(ResolverImpLangPrivateKey, ScalaLangName)
	newRule.SetPrivateAttr(config.GazelleImportsKey, s.imports())
	newRule.SetPrivateAttr(scalaImportsPrivateKey, s.imports())

	return newRule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaLibraryRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	if imps, ok := r.PrivateAttr(scalaImportsPrivateKey).([]string); ok {
		specs := make([]resolve.ImportSpec, len(imps))
		for i, imp := range imps {
			specs[i] = resolve.ImportSpec{
				Lang: "scala",
				Imp:  imp,
			}
		}
		return specs
	}
	return nil
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaLibraryRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
	resolveDeps("deps")(c, ix, r, imports, from)
}
