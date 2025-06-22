package semanticdb

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	ScalaFilesRuleKind = "scala_files"
	ScalaFilesRuleLoad = "@build_stack_scala_gazelle//rules:scala_files.bzl"
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		provider := NewScalaFilesRuleProvider(load, kind)
		if err := scalarule.GlobalProviderRegistry().RegisterProvider(fqn, provider); err != nil {
			log.Fatalf("registering %s rule provider: %v", ScalaFilesRuleKind, err)
		}
	}
	mustRegister(ScalaFilesRuleLoad, ScalaFilesRuleKind)
}

func NewScalaFilesRuleProvider(load, kind string) *ScalaFilesRuleProvider {
	return &ScalaFilesRuleProvider{load, kind}
}

// ScalaFilesRuleProvider implements a scalarule.Provider for the semanticdb_index.
type ScalaFilesRuleProvider struct {
	load, kind string
}

// Name implements part of the scalarule.Provider interface.
func (s *ScalaFilesRuleProvider) Name() string {
	return s.kind
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *ScalaFilesRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		NonEmptyAttrs: map[string]bool{
			"srcs": true,
		},
		MergeableAttrs: map[string]bool{
			"srcs": true,
		},
		ResolveAttrs: map[string]bool{
			"srcs": true,
		},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *ScalaFilesRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.kind},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.
func (s *ScalaFilesRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	r := rule.NewRule(s.kind, s.kind)
	r.SetAttr("tags", []string{"manual"})
	r.SetAttr("visibility", []string{"//visibility:public"})
	r.SetPrivateAttr(config.GazelleImportsKey, []string{})

	return &scalaFilesRule{cfg, pkg, r}
}

// ResolveRule implements the RuleResolver interface. It always
// returns nil.  The ProvideRule interface is the intended use case.
func (s *ScalaFilesRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {
	return nil
}

// scalaFilesRule implements scalarule.RuleProvider for existing scala rules.
type scalaFilesRule struct {
	cfg  *scalarule.Config
	pkg  scalarule.Package
	rule *rule.Rule
}

// Kind implements part of the ruleProvider interface.
func (s *scalaFilesRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaFilesRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaFilesRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the scalarule.RuleProvider interface.  It retuns a
// non-nil struct in order to be scheduled by gazelle to resolve.
func (s *scalaFilesRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return []resolve.ImportSpec{}
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *scalaFilesRule) Resolve(rctx *scalarule.ResolveContext, importsRaw interface{}) {
	srcs := rctx.Rule.AttrStrings("srcs")

	rel := s.pkg.GenerateArgs().Rel
	for _, file := range s.pkg.Files() {
		relativeToPkg := strings.TrimPrefix(file.Filename, rel)
		srcs = append(srcs, strings.TrimPrefix(relativeToPkg, "/"))
	}

	if len(srcs) > 0 {
		rctx.Rule.SetAttr("srcs", collections.DeduplicateAndSort(srcs))
		filesets.Add(rctx.From)
	} else {
		rctx.Rule.Delete()
	}
}
