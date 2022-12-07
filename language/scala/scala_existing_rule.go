package scala

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/rules_proto/pkg/protoc"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/glob"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func init() {
	mustRegister := func(load, kind string, isBinaryRule bool) {
		fqn := load + "%" + kind
		Rules().MustRegisterRule(fqn, &scalaExistingRule{load, kind, isBinaryRule})
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary", true)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library", false)
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test", true)
}

// scalaExistingRule implements RuleResolver for scala-kind rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is used to parse files named in 'srcs' and update 'deps'.
type scalaExistingRule struct {
	load, name   string
	isBinaryRule bool
}

// Name implements part of the RuleInfo interface.
func (s *scalaExistingRule) Name() string {
	return s.name
}

// KindInfo implements part of the RuleInfo interface.
func (s *scalaExistingRule) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the RuleInfo interface.
func (s *scalaExistingRule) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    s.load,
		Symbols: []string{s.name},
	}
}

// ProvideRule implements part of the RuleInfo interface.  It always returns
// nil.  The ResolveRule interface is the intended use case.
func (s *scalaExistingRule) ProvideRule(cfg *RuleConfig, pkg ScalaPackage) RuleProvider {
	return nil
}

// ResolveRule implements the RuleResolver interface.
func (s *scalaExistingRule) ResolveRule(cfg *RuleConfig, pkg ScalaPackage, r *rule.Rule) RuleProvider {
	filenames, err := glob.CollectFilenames(pkg.File(), pkg.Dir(), pkg.Rel(), r.Attr("srcs"))
	if err != nil {
		log.Printf("skipping %s //%s:%s (unable to collect srcs: %v)", r.Kind(), pkg.Rel(), r.Name(), err)
		return nil
	}

	from := label.New("", pkg.Rel(), r.Name())
	files := make([]*sppb.File, 0)

	if len(filenames) > 0 {
		files, err = parseScalaFiles(pkg.Dir(), from, r.Kind(), filenames, pkg.ScalaParser())
		if err != nil {
			log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
			return nil
		}
	}

	r.SetPrivateAttr(config.GazelleImportsKey, files)
	r.SetPrivateAttr(resolverImpLangPrivateKey, "java")
	// r.SetPrivateAttr(resolverImpLangPrivateKey, ScalaLangName)

	return &scalaExistingRuleProvider{cfg, pkg, r, files, s.isBinaryRule}
}

// scalaExistingRuleProvider implements RuleProvider for existing scala rules.
type scalaExistingRuleProvider struct {
	cfg          *RuleConfig
	pkg          ScalaPackage
	rule         *rule.Rule
	files        []*sppb.File
	isBinaryRule bool
}

// Kind implements part of the ruleProvider interface.
func (s *scalaExistingRuleProvider) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaExistingRuleProvider) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaExistingRuleProvider) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaExistingRuleProvider) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	// binary rules are not deps of anything else, so we don't advertise to
	// provide any imports
	if s.isBinaryRule {
		return nil
	}

	// set the impLang to a default value.  If there is a map_kind_import_name
	// associated with this kind, return that instead.  This should force the
	// ruleIndex to miss on the impLang, allowing us to override in the source
	// CrossResolver.
	sc := getScalaConfig(c)
	lang := scalaLangName
	if _, ok := sc.labelNameRewrites[r.Kind()]; ok {
		lang = r.Kind()
	}

	provides := make([]string, 0)
	for _, file := range s.files {
		provides = append(provides, file.Packages...)
		provides = append(provides, file.Classes...)
		provides = append(provides, file.Objects...)
		provides = append(provides, file.Traits...)
		provides = append(provides, file.Types...)
		provides = append(provides, file.Vals...)
	}
	provides = protoc.DeduplicateAndSort(provides)

	specs := make([]resolve.ImportSpec, len(provides))
	for i, imp := range provides {
		specs[i] = resolve.ImportSpec{Lang: lang, Imp: imp}
		// log.Println("scalaExistingRule.Imports()", lang, r.Kind(), r.Name(), i, imp)
	}

	return specs
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaExistingRuleProvider) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
	files, ok := importsRaw.([]*sppb.File)
	if !ok {
		return
	}

	sc := getScalaConfig(c)
	imports := collectImports(sc, from, r, files)

	if len(imports) > 0 {
		for _, imp := range imports.Values() {
			if known, err := sc.ResolveKnownImport(c, ix, from, scalaLangName, imp.Imp); err != nil {
				imp.Error = err
			} else {
				imp.Known = known
			}
		}

		deps := buildKeepDepsList(sc, r.Attr("deps"))
		addResolvedDeps(deps, sc, r.Kind(), from, imports)

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
		if !(wantImports || (wantUnresolved && imp.Known == nil)) {
			continue
		}
		var impType string
		if imp.Known != nil {
			impType = fmt.Sprintf("%v", imp.Known.Type)
		} else if imp.Error != nil {
			impType = "ERROR"
		}
		parts := []string{
			fmt.Sprintf("# %s<%s>", key, impType),
		}

		if imp.Known != nil {
			to := imp.Known.Label.String()
			if to == "//:" {
				to = "NO-LABEL"
			}
			parts = append(parts, fmt.Sprintf("✅ %s<%s>", to, imp.Known.Provider))
		} else if imp.Error != nil {
			parts = append(parts, fmt.Sprintf("❌ %v", imp.Error))
		}

		parts = append(parts, fmt.Sprintf("(%v", imp.Kind))
		if imp.Source != nil {
			parts = append(parts, fmt.Sprintf("of %s)", filepath.Base(imp.Source.Filename)))
		} else if imp.Src != "" {
			parts = append(parts, fmt.Sprintf("of %s)", imp.Src))
		}

		comments.Before = append(comments.Before, build.Comment{Token: strings.Join(parts, " ")})
	}

	// imports.Annotate(comments, func(i *resolver.Import) bool {
	// 	if wantImports {
	// 		return true
	// 	}
	// 	if wantUnresolved || i.Error != nil {
	// 		return true
	// 	}
	// 	return false
	// })
}
