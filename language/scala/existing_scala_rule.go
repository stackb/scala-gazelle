package scala

import (
	"bufio"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

func init() {
	mustRegister := func(load, kind string, isBinary, isLibrary, isTest bool) {
		fqn := load + "%" + kind
		if err := scalarule.
			GlobalProviderRegistry().
			RegisterProvider(fqn, &existingScalaRuleProvider{load, kind, isBinary, isLibrary, isTest}); err != nil {
			log.Fatalf("registering scala_rule providers: %v", err)
		}
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary", true, false, false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library", false, true, false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library", false, true, false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test", false, false, true)
}

// existingScalaRuleProvider implements RuleResolver for scala-like rules that
// are already in the build file.  It does not create any new rules.  This rule
// implementation is used to parse files named in 'srcs' and update 'deps' (and
// optionally, exports).
type existingScalaRuleProvider struct {
	load, name string
	isBinary   bool
	isLibrary  bool
	isTest     bool
}

// Name implements part of the scalarule.Provider interface.
func (s *existingScalaRuleProvider) Name() string {
	return s.name
}

// KindInfo implements part of the scalarule.Provider interface.
func (s *existingScalaRuleProvider) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the scalarule.Provider interface.
func (s *existingScalaRuleProvider) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the scalarule.Provider interface.  It always
// returns nil.  The ResolveRule interface is the intended use case.
func (s *existingScalaRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *existingScalaRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {
	scalaRule, err := pkg.ParseRule(r, "srcs")
	if err != nil {
		if err == ErrRuleHasNoSrcs {
			return nil // no need to print a warning
		}
		log.Printf("skipping %s %s: unable to collect srcs: %v", r.Kind(), r.Name(), err)
		return nil
	}
	if scalaRule == nil {
		log.Panicln("scalaRule should not be nil!")
	}

	r.SetPrivateAttr(config.GazelleImportsKey, scalaRule)

	return &existingScalaRule{cfg, pkg, r, scalaRule, s.isBinary, s.isLibrary, s.isTest}
}

// existingScalaRule implements scalarule.RuleProvider for existing scala rules.
type existingScalaRule struct {
	cfg       *scalarule.Config
	pkg       scalarule.Package
	rule      *rule.Rule
	scalaRule scalarule.Rule
	isBinary  bool
	isLibrary bool
	isTest    bool
}

// Kind implements part of the ruleProvider interface.
func (s *existingScalaRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *existingScalaRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *existingScalaRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the scalarule.RuleProvider interface.
func (s *existingScalaRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	return s.scalaRule.Provides()
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *existingScalaRule) Resolve(rctx *scalarule.ResolveContext, importsRaw interface{}) {
	scalaRule, ok := importsRaw.(*scalaRule)
	if !ok {
		return
	}

	sc := scalaconfig.Get(rctx.Config)
	sc.Imports(scalaRule.ResolveImports(rctx), rctx.Rule, "deps", rctx.From)
	if s.isLibrary {
		sc.Exports(scalaRule.ResolveExports(rctx), rctx.Rule, "exports", rctx.From)
	}

	srcs := rctx.Rule.Attr("srcs")
	if sc.ShouldAnnotateRule() {
		if comments, err := makeRuleComments(scalaRule.pb); err != nil {
			log.Fatalln("annotating rule:", err)
		} else {
			srcs.Comment().Before = comments
		}
	} else {
		srcs.Comment().Before = nil
	}
}

func makeRuleComments(pb *sppb.Rule) (comments []build.Comment, err error) {
	pb.ParseTimeMillis = 0
	json, err := protobuf.PrettyText(pb)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(json))
	for scanner.Scan() {
		line := scanner.Text()
		comments = append(comments, build.Comment{
			Token: "# " + line,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}
