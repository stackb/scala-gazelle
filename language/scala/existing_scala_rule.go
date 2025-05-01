package scala

import (
	"bufio"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	"github.com/stackb/scala-gazelle/pkg/bazel"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/stackb/scala-gazelle/pkg/scalarule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
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
		if err != ErrRuleHasNoSrcs {
			log.Printf("skipping %s %s: unable to collect srcs: %v", r.Kind(), r.Name(), err)
			return nil
		}
		// rule has no srcs.  This is OK for binary rules, sometimes they only
		// have a main_class.
		// if !s.isBinary {
		// 	return nil // no need to print a warning
		// }
	}
	if scalaRule == nil {
		log.Panicln("scalaRule should not be nil!")
	}

	r.SetPrivateAttr(config.GazelleImportsKey, scalaRule)
	r.SetPrivateAttr("_scala_files", scalaRule.Files())

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
	imports := scalaRule.ResolveImports(rctx)
	sc.Imports(imports, rctx.Rule, "deps", rctx.From)

	commentsSrcs := rctx.Rule.AttrComments("srcs")
	if commentsSrcs != nil {
		commentsSrcs.Before = nil
		if sc.ShouldAnnotateImports() {
			scalaconfig.AnnotateImports(imports, commentsSrcs, "import: ")
		}
		if sc.ShouldAnnotateRule() {
			ruleComments := makeRuleComments(scalaRule.pb)
			commentsSrcs.Before = append(commentsSrcs.Before, ruleComments...)
		}
	}

	if s.isLibrary {
		exports := scalaRule.ResolveExports(rctx)
		sc.Exports(exports, rctx.Rule, "exports", rctx.From)
	}

	if sc.ShouldSweepTransitiveDeps() {
		if err := s.sweepTransitive("deps", rctx.Rule, rctx.From); err != nil {
			log.Panicf("transitive sweep failed: %v", err)
		}
	}

}

// sweepTransitive iterates through deps marked "TRANSITIVE" and removes them if
// the target still builds without it.
func (s *existingScalaRule) sweepTransitive(attrName string, r *rule.Rule, from label.Label) error {
	expr := r.Attr(attrName)
	if expr == nil {
		return nil
	}

	deps, isList := expr.(*build.ListExpr)
	if !isList {
		return nil // some other condition we can't deal with
	}

	file := s.pkg.GenerateArgs().File

	for i := len(deps.List) - 1; i >= 0; i-- {
		expr := deps.List[i]
		switch t := expr.(type) {
		case *build.StringExpr:
			if len(t.Comments.Suffix) != 1 {
				continue
			}
			if t.Comments.Suffix[0].Token != "# TRANSITIVE" {
				continue
			}

			dep, err := label.Parse(t.Value)
			if err != nil {
				return err
			}
			deps.List = collections.SliceRemoveIndex(deps.List, i)

			if err := file.Save(file.Path); err != nil {
				return err
			}

			if _, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode == 0 {
				log.Println("🗑️ transitive junk: ", dep, file.Path)
			} else {
				// log.Println("👑 transitive keep: ", dep, file.Path)
				deps.List = collections.SliceInsertAt(deps.List, i, expr)
			}
		}

	}

	if err := file.Save(file.Path); err != nil {
		return err
	}

	return nil
}

func makeRuleComments(pb *sppb.Rule) (comments []build.Comment) {
	pb.ParseTimeMillis = 0
	json, _ := protobuf.StableJSON(pb) // ignoring error, this isn't critical

	scanner := bufio.NewScanner(strings.NewReader(json))
	for scanner.Scan() {
		line := scanner.Text()
		comments = append(comments, build.Comment{
			Token: "# " + line,
		})
	}
	return
}
