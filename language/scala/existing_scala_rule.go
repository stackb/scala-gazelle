package scala

import (
	"bufio"
	"fmt"
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
}

// sweepTransitiveDeps iterates through deps marked "UNKNOWN" and removes them
// if the target still builds without it.
func (s *existingScalaRule) sweepTransitiveAttr(attrName string, r *rule.Rule, from label.Label) ([]string, error) {
	// get the File to which this Rule belongs
	file := s.pkg.GenerateArgs().File

	return s.sweepTransitive(attrName, file, r, from)
}

// sweepTransitive iterates through deps marked "UNKNOWN" and removes them if
// the target still builds without it.
func (s *existingScalaRule) sweepTransitive(attrName string, file *rule.File, r *rule.Rule, from label.Label) (junk []string, err error) {
	expr := r.Attr(attrName)
	if expr == nil {
		return nil, nil
	}

	deps, isList := expr.(*build.ListExpr)
	if !isList {
		return nil, nil // some other condition we can't deal with
	}

	// check that the deps have at least one unknown dep
	var hasTransitiveDeps bool
	for _, expr := range deps.List {
		if str, ok := expr.(*build.StringExpr); ok {
			for _, suffix := range str.Comment().Suffix {
				if suffix.Token == scalaconfig.TransitiveCommentToken {
					hasTransitiveDeps = true
					break
				}
			}
		}
	}
	if !hasTransitiveDeps {
		return nil, nil // nothing to do
	}

	// target should build first time, otherwise we can't check accurately.
	log.Println("🧱 transitive sweep:", from)

	if out, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode != 0 {
		log.Fatalln("sweep failed (must build cleanly on first attempt): %s", string(out))
	}

	// iterate the list backwards
	for i := len(deps.List) - 1; i >= 0; i-- {
		expr := deps.List[i]

		// look for transitive string dep expressions
		dep, ok := expr.(*build.StringExpr)
		if !ok {
			continue
		}
		var isTransitiveDep bool
		for _, suffix := range dep.Comment().Suffix {
			if suffix.Token == scalaconfig.TransitiveCommentToken {
				isTransitiveDep = true
				break
			}
		}
		if !isTransitiveDep {
			continue
		}

		// reference of original list in case it does not build
		original := deps.List
		// reset deps with this one spliced out
		deps.List = collections.SliceRemoveIndex(deps.List, i)
		// save file to reflect change
		if err := file.Save(file.Path); err != nil {
			return nil, err
		}
		// see if it still builds
		if _, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode == 0 {
			log.Println("- 💩 junk:", dep.Value)
			junk = append(junk, dep.Value)
		} else {
			log.Println("- 👑 keep:", dep.Value)
			deps.List = original
		}
	}

	// final save with possible last change
	if err := file.Save(file.Path); err != nil {
		return nil, err
	}

	return
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

func formatBuildozerRemoveDeps(from label.Label, junk []string) string {
	return fmt.Sprintf("buildozer 'remove deps %s' %s", strings.Join(junk, " "), from.String())
}

func hasTransitiveComment(r *rule.Rule) bool {
	for _, before := range r.Comments() {
		if before == scalaconfig.TransitiveCommentToken {
			return true
		}
	}
	return false
}
