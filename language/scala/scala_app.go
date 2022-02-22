package scala

import (
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/rules_proto/pkg/protoc"
)

func init() {
	Rules().MustRegisterRule("stackb:rules_proto:scala_app",
		&scalaExistingRule{"//bazel_tools.bzl/scala:scala.bzl", "scala_app"})

	Rules().MustRegisterRule("stackb:rules_proto:scala_app_test",
		&scalaExistingRule{"//bazel_tools.bzl/scala:scala.bzl", "scala_app_test"})

	Rules().MustRegisterRule("stackb:rules_proto:scala_library",
		&scalaExistingRule{"@io_bazel_rules_scala//scala:scala.bzl", "scala_library"})

	Rules().MustRegisterRule("stackb:rules_proto:scala_binary",
		&scalaExistingRule{"@io_bazel_rules_scala//scala:scala.bzl", "scala_binary"})

	Rules().MustRegisterRule("stackb:rules_proto:scala_test",
		&scalaExistingRule{"@io_bazel_rules_scala//scala:scala.bzl", "scala_test"})
}

// scalaExistingRule implements RuleResolver for scala-kind rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is to parse source files and update deps.
type scalaExistingRule struct{ load, name string }

// Name implements part of the RuleInfo interface.
func (s *scalaExistingRule) Name() string {
	return s.name
}

// KindInfo implements part of the RuleInfo interface.
func (s *scalaExistingRule) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		// TODO(pcj): understand better why deps needs to be in MergeableAttrs
		// here rather than ResolveAttrs.
		MergeableAttrs: map[string]bool{
			"srcs": true,
			"deps": true,
		},
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

// ResolveRule implement the RuleResolver interface.  It will attempt to parse
// imports and resolve deps.
func (s *scalaExistingRule) ResolveRule(cfg *RuleConfig, pkg ScalaPackage, existing *rule.Rule) RuleProvider {
	// If we cannot find any srcs for the rule, bail now.
	srcs := getAttrFiles(pkg, existing, "srcs")
	if len(srcs) == 0 {
		return nil
	}

	resolver, err := CrossResolvers().LookupCrossResolver("stackb:scala-gazelle:scala-source-index")
	if err != nil {
		log.Fatal("unable to find scala source cross resolver!")
	}

	from := label.New("", pkg.Rel(), existing.Name())

	requires, provides := resolveSrcsSymbols(pkg.Dir(), from, srcs, resolver.(*scalaSourceIndexResolver))

	existing.SetPrivateAttr(config.GazelleImportsKey, requires)
	existing.SetPrivateAttr(ResolverImpLangPrivateKey, "scala")

	if debug {
		for _, imp := range requires {
			existing.AddComment("# import: " + imp)
		}
	}

	return &scalaExistingRuleRule{cfg, pkg, existing, requires, provides}
}

// scalaExistingRuleRule implements RuleProvider for existing scala rules.
type scalaExistingRuleRule struct {
	cfg  *RuleConfig
	pkg  ScalaPackage
	rule *rule.Rule
	// requires is the list of scala symbols this group of files requires
	// (import statements).
	requires []string
	// provides is the list of scala symbols this group of files provides
	// (classes, traits, etc).
	provides []string
}

// Kind implements part of the ruleProvider interface.
func (s *scalaExistingRuleRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaExistingRuleRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaExistingRuleRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaExistingRuleRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	specs := make([]resolve.ImportSpec, len(s.provides))
	for i, imp := range s.provides {
		specs[i] = resolve.ImportSpec{
			Lang: "scala",
			Imp:  imp,
		}
	}
	return specs
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaExistingRuleRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
	resolveDeps("deps")(c, ix, r, imports, from)
}

// getAttrFiles returns a list of source files for the 'srcs' attribute.  Each
// value is a repo-relative path.
func getAttrFiles(pkg ScalaPackage, r *rule.Rule, attrName string) (srcs []string) {
	switch t := r.Attr(attrName).(type) {
	case *build.ListExpr:
		// probably ["foo.scala", "bar.scala"]
		for _, item := range t.List {
			switch elem := item.(type) {
			case *build.StringExpr:
				value := elem.Token
				srcs = append(srcs, value)
			}
		}
	case *build.CallExpr:
		// probably glob(["**/*.scala"])
		ident, ok := t.X.(*build.Ident)
		if !ok {
			break
		}
		switch ident.Name {
		case "glob":
			glob := parseGlob(t)
			dir := filepath.Join(pkg.Dir(), pkg.Rel())
			srcs = append(srcs, applyGlob(glob, os.DirFS(dir))...)
		default:
			log.Println("ignoring srcs call expression: %+v", t)
		}
	default:
		log.Printf("unknown srcs types: //%s:%s %T", pkg.Rel(), r.Name(), t)
	}

	return
}

func parseScalaFiles(dir string, files []string) []*ScalaFile {
	scalaFiles := make([]*ScalaFile, 0)
	for _, file := range files {
		f, err := ParseScalaFile(dir, file)
		if err != nil {
			log.Println("error parsing scala file", file, err)
			continue
		}
		scalaFiles = append(scalaFiles, f)
	}
	return scalaFiles
}

func scalaImports(files []*ScalaFile) []string {
	imports := make([]string, 0)
	for _, f := range files {
		for _, imp := range f.Imports {
			imports = append(imports, imp.Name)
		}
	}
	return imports
}

func resolveSrcsSymbols(dir string, from label.Label, srcs []string, resolver *scalaSourceIndexResolver) (requires, provides []string) {
	spec, err := resolver.ParseScalaRuleSpec(dir, from, srcs...)
	if err != nil {
		log.Println("failed to parse scala sources", from, err)
	}

	for _, file := range spec.Srcs {
		requires = append(requires, file.Imports...)
		provides = append(provides, file.Packages...)
		provides = append(provides, file.Classes...)
		provides = append(provides, file.Objects...)
		provides = append(provides, file.Traits...)
	}

	requires = protoc.DeduplicateAndSort(requires)
	provides = protoc.DeduplicateAndSort(provides)
	return
}
