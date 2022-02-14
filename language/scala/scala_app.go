package scala

import (
	"log"
	"os"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bmatcuk/doublestar"
)

func init() {
	Rules().MustRegisterRule("stackb:rules_proto:scala_app", &scalaApp{})
}

// scalaApp implements RuleResolver for the 'scala_app' rule from
// @rules_scala.
type scalaApp struct{}

// Name implements part of the RuleInfo interface.
func (s *scalaApp) Name() string {
	return "scala_app"
}

// KindInfo implements part of the RuleInfo interface.
func (s *scalaApp) KindInfo() rule.KindInfo {
	return rule.KindInfo{
		MergeableAttrs: map[string]bool{
			"srcs": true,
		},
		ResolveAttrs: map[string]bool{"deps": true},
	}
}

// LoadInfo implements part of the RuleInfo interface.
func (s *scalaApp) LoadInfo() rule.LoadInfo {
	return rule.LoadInfo{
		Name:    "//bazel_tools.bzl/scala:scala.bzl",
		Symbols: []string{"scala_app"},
	}
}

// ProvideRule implements part of the RuleInfo interface.  It always returns
// nil.  The ResolveRule interface is the intended use case.
func (s *scalaApp) ProvideRule(cfg *RuleConfig, pkg ScalaPackage) RuleProvider {
	return nil
}

// ResolveRule implement the RuleResolver interface.  It will attempt to parse
// imports and resolve deps.
func (s *scalaApp) ResolveRule(cfg *RuleConfig, pkg ScalaPackage, existing *rule.Rule) RuleProvider {
	return &scalaAppRule{cfg, pkg, existing}
}

// scalaAppRule implements RuleProvider for 'scala_library'-derived rules.
type scalaAppRule struct {
	cfg  *RuleConfig
	pkg  ScalaPackage
	rule *rule.Rule
}

// Kind implements part of the ruleProvider interface.
func (s *scalaAppRule) Kind() string {
	return s.rule.Kind()
}

// Name implements part of the ruleProvider interface.
func (s *scalaAppRule) Name() string {
	return s.rule.Name()
}

// Rule implements part of the ruleProvider interface.
func (s *scalaAppRule) Rule() *rule.Rule {
	return s.rule
}

// Imports implements part of the RuleProvider interface.
func (s *scalaAppRule) Imports(c *config.Config, r *rule.Rule, file *rule.File) []resolve.ImportSpec {
	log.Printf("resolving imports!")

	srcs := s.getSrcsFiles()

	// if imps, ok := r.PrivateAttr(scalaImportsPrivateKey).([]string); ok {
	// 	specs := make([]resolve.ImportSpec, len(imps))
	// 	for i, imp := range imps {
	// 		specs[i] = resolve.ImportSpec{
	// 			Lang: "scala",
	// 			Imp:  imp,
	// 		}
	// 	}
	// 	return specs
	// }
	return nil
}

// getSrcsFiles returns a list of source files for the 'srcs' attribute.  Each
// value is a repo-relative path.
func (s *scalaAppRule) getSrcsFiles() (srcs []string) {
	files := make([]*ScalaFile, 0)

	switch t := s.rule.Attr("srcs").(type) {
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
			fs := os.DirFS(s.pkg.Rel())
			for _, pattern := range glob.Patterns {
				names, err := doublestar.Glob(fs, pattern)
				if err != nil {
					// doublestar.Match returns only one possible error, and only if the
					// pattern is not valid. During the configuration of the walker (see
					// Configure below), we discard any invalid pattern and thus an error
					// here should not be possible.
					log.Printf("error during doublestar.Glob: %v (pattern ignored: %v)", err, pattern)
					continue
				}
				srcs = append(srcs, names...)
			}
		default:
			log.Println("ignoring srcs call expression: %+v", t)
		}
	default:
		log.Printf("unknown srcs types: %T", t)
	}

	return
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaAppRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, imports []string, from label.Label) {
	// resolveDeps("deps")(c, ix, r, imports, from)
}

func parseGlob(call *build.CallExpr) (glob rule.GlobValue) {
	log.Printf("parsing glob! %+v", call)

	for _, expr := range call.List {
		switch list := expr.(type) {
		case *build.ListExpr:
			for _, item := range list.List {
				switch elem := item.(type) {
				case *build.StringExpr:
					value := elem.Token
					glob.Patterns = append(glob.Patterns, value)
				default:
					log.Printf("skipping glob list item expression: %+v (%T)", elem, elem)
				}
			}
		default:
			log.Println("skipping glob list expression: %+v", expr)
		}
	}

	return
}
