package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

func init() {
	mustRegister := func(load, kind string, isBinaryRule bool) {
		fqn := load + "%" + kind
		if err := scalarule.
			GlobalProviderRegistry().
			RegisterProvider(fqn, &existingScalaRuleProvider{load, kind, isBinaryRule}); err != nil {
			log.Fatalf("registering scala_rule providers: %v", err)
		}
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary", true)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test", true)
}

// existingScalaRuleProvider implements RuleResolver for scala-like rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is used to parse files named in 'srcs' and update 'deps'.
type existingScalaRuleProvider struct {
	load, name   string
	isBinaryRule bool
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

// ProvideRule implements part of the scalarule.Provider interface.  It always returns
// nil.  The ResolveRule interface is the intended use case.
func (s *existingScalaRuleProvider) ProvideRule(cfg *scalarule.Config, pkg scalarule.Package) scalarule.RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *existingScalaRuleProvider) ResolveRule(cfg *scalarule.Config, pkg scalarule.Package, r *rule.Rule) scalarule.RuleProvider {
	scalaRule, err := pkg.ParseRule(r, "srcs")
	if err != nil {
		log.Printf("skipping %s %s: unable to collect srcs: %v", r.Kind(), r.Name(), err)
		return nil
	}
	if scalaRule == nil {
		log.Panicln("scalaRule should not be nil!")
	}

	r.SetPrivateAttr(config.GazelleImportsKey, scalaRule)

	return &existingScalaRule{cfg, pkg, r, scalaRule, s.isBinaryRule}
}

// existingScalaRule implements scalarule.RuleProvider for existing scala rules.
type existingScalaRule struct {
	cfg          *scalarule.Config
	pkg          scalarule.Package
	rule         *rule.Rule
	scalaRule    scalarule.Rule
	isBinaryRule bool
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
	// binary rules should not be deps of anything else, so we don't advertise
	// any imports. TODO(pcj): this is too simplisitic: test helpers can be used
	// by other test rules.  So we should probably use the 'impLang' to differentiate
	// generic imports ('scala') vs testonly imports ('test').
	if s.isBinaryRule {
		return nil
	}
	return s.scalaRule.Exports()
}

// Resolve implements part of the scalarule.RuleProvider interface.
func (s *existingScalaRule) Resolve(ctx *scalarule.ResolveContext, importsRaw interface{}) {
	scalaRule, ok := importsRaw.(*scalaRule)
	if !ok {
		return
	}

	r := ctx.Rule
	sc := getScalaConfig(ctx.Config)
	imports := scalaRule.Imports()

	if len(imports) > 0 {
		for _, imp := range imports.Values() {
			if symbol, err := scalaRule.ResolveSymbol(ctx.Config, ctx.RuleIndex, ctx.From, scalaLangName, imp.Imp); err != nil {
				imp.Error = err
			} else {
				if len(symbol.Conflicts) > 0 {
					files := scalaRule.Files()
					filenames := make([]string, len(files))
					for i, file := range files {
						filenames[i] = file.Filename
					}
					// if resp, err := ctx.Compiler.CompileScala(ctx.Config.RepoRoot, filenames); err != nil {
					// 	log.Println("scala compiler error:", err)
					// } else {
					// 	if false {
					// 		// log.Printf("scala compiler response: %+v", resp)
					// 		for _, nf := range resp.NotFound {
					// 			log.Println("not found: ", nf.Kind, nf.Name)
					// 		}
					// 		for _, nm := range resp.NotMember {
					// 			log.Println("not member: ", nm.Kind, nm.Name, nm.Package)
					// 		}
					// 	}
					// }
					// if false {
					// 	log.Printf("conflicting symbol resolution for %v %q:", symbol.Type, imp.Imp)
					// 	log.Println(" - choose one of the following to suppress this message:")
					// 	log.Printf("    # gazelle:resolve scala %s %s", imp.Imp, symbol.Label)
					// 	for _, conflict := range symbol.Conflicts {
					// 		log.Printf("# gazelle:resolve scala %s %s", imp.Imp, conflict.Label)
					// 	}
					// }
				}
				imp.Symbol = symbol
			}
		}

		deps := buildKeepDepsList(sc, r.Attr("deps"))
		addResolvedDeps(deps, sc, r.Kind(), ctx.From, imports)

		r.SetAttr("deps", deps)
	}

	if sc.shouldAnnotateImports() || sc.shouldAnnotateResolvedDeps() {
		attr := r.Attr("srcs")
		switch t := attr.(type) {
		case *build.ListExpr:
			annotateImports(imports, &t.Comments, sc.shouldAnnotateImports(), sc.shouldAnnotateUnresolvedDeps())
		case *build.CallExpr:
			annotateImports(imports, &t.Comments, sc.shouldAnnotateImports(), sc.shouldAnnotateUnresolvedDeps())
		case *build.BinaryExpr:
			annotateImports(imports, &t.Comments, sc.shouldAnnotateImports(), sc.shouldAnnotateUnresolvedDeps())
		}
	}
}

func annotateImports(imports resolver.ImportMap, comments *build.Comments, wantImports, wantUnresolved bool) {
	comments.Before = nil
	for _, key := range imports.Keys() {
		imp := imports[key]
		if !(wantImports || (wantUnresolved && imp.Symbol == nil)) {
			continue
		}
		comments.Before = append(comments.Before, imp.Comment())
	}
}
