package scala

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/rules_proto/pkg/protoc"
	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	mustRegister := func(load, kind string) {
		fqn := load + "%" + kind
		Rules().MustRegisterRule(fqn, &scalaExistingRule{load, kind})
	}

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_macro_library")
	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "scala_test")

	mustRegister("@io_bazel_rules_scala//scala:scala.bzl", "_scala_library")
	mustRegister("//bazel_tools:scala.bzl", "scala_app")
	mustRegister("//bazel_tools:scala.bzl", "scala_app_test")
	mustRegister("//bazel_tools:scala.bzl", "scala_app_library")
	mustRegister("//bazel_tools:scala.bzl", "trumid_scala_library")
	mustRegister("//bazel_tools:scala.bzl", "trumid_scala_test")
	mustRegister("//bazel_tools:scala.bzl", "classic_scala_app")
	mustRegister("//bazel_tools:scala.bzl", "scala_e2e_app")
	mustRegister("//bazel_tools:scala.bzl", "scala_e2e_test")
}

// scalaExistingRule implements RuleResolver for scala-kind rules that are
// already in the build file.  It does not create any new rules.  This rule
// implementation is to parse files named in 'srcs' and update 'deps'.
type scalaExistingRule struct {
	load, name string
}

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
			"deps": true,
		},
		ResolveAttrs: map[string]bool{
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
func (s *scalaExistingRule) ResolveRule(cfg *RuleConfig, pkg ScalaPackage, r *rule.Rule) RuleProvider {
	// TODO(pcj): is this rule copying actually necessary?  Introduced this when
	// trying to debug a bug where I wasn't sure if state inside the rule was
	// the issue.
	// r := rule.NewRule(existing.Kind(), existing.Name())
	// for _, key := range existing.AttrKeys() {
	// 	r.SetAttr(key, existing.Attr(key))
	// }
	// r.DelAttr("deps") // make sure the "source" rule has no deps to start

	srcs, err := getAttrFiles(pkg, r, "srcs")
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
		return nil
	}

	// If we cannot find any srcs for the rule, skip it.
	if len(srcs) == 0 {
		log.Printf("skipping %s //%s:%s (no srcs)", r.Kind(), pkg.Rel(), r.Name())
		return nil
	}

	from := label.New("", pkg.Rel(), r.Name())

	files, err := resolveScalaSrcs(pkg.Dir(), from, r.Kind(), srcs, pkg.ScalaFileParser())
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
		return nil
	}

	r.SetPrivateAttr(config.GazelleImportsKey, files)
	r.SetPrivateAttr(ResolverImpLangPrivateKey, "scala")

	return &scalaExistingRuleRule{cfg, pkg, r, files}
}

// scalaExistingRuleRule implements RuleProvider for existing scala rules.
type scalaExistingRuleRule struct {
	cfg   *RuleConfig
	pkg   ScalaPackage
	rule  *rule.Rule
	files []*index.ScalaFileSpec
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

	// set the impLang to a default value.  If there is a map_kind_import_name
	// associated with this kind, return that instead.  This should force the
	// ruleIndex to miss on the impLang, allowing us to override in the source
	// CrossResolver.
	sc := getScalaConfig(c)
	lang := ScalaLangName
	if _, ok := sc.mapKindImportNames[r.Kind()]; ok {
		lang = r.Kind()
	}

	specs := make([]resolve.ImportSpec, len(provides))
	for i, imp := range provides {
		specs[i] = resolve.ImportSpec{
			Lang: lang,
			Imp:  imp,
		}
	}

	return specs
}

// Resolve implements part of the RuleProvider interface.
func (s *scalaExistingRuleRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
	files, ok := importsRaw.([]*index.ScalaFileSpec)
	if !ok {
		return
	}

	// Local variables
	dbg := debug
	importRegistry := s.pkg.ScalaImportRegistry()
	imports := make(map[string]*importOrigin)

	impLang := r.Kind()
	if overrideImpLang, ok := r.PrivateAttr(ResolverImpLangPrivateKey).(string); ok {
		impLang = overrideImpLang
	}

	resolved := make(labelImportMap)
	// preinit a slot for unresolved deps
	resolved[label.NoLabel] = make(map[string]*importOrigin)

	// --- Gather imports ---

	// 1: direct
	for _, file := range files {
		for _, imp := range file.Imports {
			imports[imp] = &importOrigin{Kind: "direct", SourceFile: file}
		}
	}

	// 2: explicity named in the rule comment.
	for _, imp := range getScalaImportsFromRuleComment(r) {
		if _, ok := imports[imp]; ok {
			continue
		}
		imports[imp] = &importOrigin{Kind: "rule-comment"}
	}

	// 3: transitive of 1+2.
	gatherIndirectDependencies(c, imports)
	// resolve this (mostly direct) initial set
	resolveImports(c, ix, importRegistry, impLang, r.Kind(), from, imports, resolved)
	// resolve transitive set
	resolveTransitive(c, ix, importRegistry, impLang, r.Kind(), from, imports, resolved)

	unresolved := resolved[label.NoLabel]
	if len(unresolved) > 0 {
		panic(fmt.Sprintf("%v has unresolved dependencies: %v", from, unresolved))
	}

	r.DelAttr("deps")
	if len(resolved) > 0 {
		r.SetAttr("deps", makeLabeledListExpr(c, r.Kind(), from, resolved))
	}

	if dbg {
		log.Println("-- | ", from, "finished deps resolution.")
		printRules(r)
	}
}

// func doExports() {
// // TODO(pcj): make this configurable
// if false && strings.Contains(r.Kind(), "library") {
// 	exported := make(map[string]*importOrigin)
// 	resolveAny := importRegistry.ResolveName
// 	resolveFromImports := resolveNameInLabelImportMap(resolved)
// 	for _, file := range files {
// 		resolve1p := resolveNameInFile(file)
// 		fileExports, err := scalaExportSymbols(file, []NameResolver{resolveFromImports, resolve1p, resolveAny})
// 		if err != nil {
// 			log.Panicf("failed to resolve export symbols: %v", err)
// 		}
// 		for _, export := range fileExports {
// 			exported[export] = &importOrigin{Kind: "export", SourceFile: file}
// 		}
// 	}
// 	r.DelAttr("exports")
// 	if len(exported) > 0 {
// 		exports := make(labelImportMap)
// 		exportKeys := make([]string, len(exported))
// 		i := 0
// 		for key := range exported {
// 			exportKeys[i] = key
// 			i++
// 		}
// 		sort.Strings(exportKeys)
// 		for _, exp := range exportKeys {
// 			origin := exported[exp]
// 			if from, ok := importRegistry.ResolveLabel(exp); ok {
// 				if from == PlatformLabel || from == label.NoLabel {
// 					continue
// 				}
// 				if has, ok := exports[from]; ok {
// 					has[exp] = origin
// 				} else {
// 					exports[from] = map[string]*importOrigin{exp: origin}
// 				}
// 			}
// 		}
// 		r.SetAttr("exports", makeLabeledListExpr(c, from, exports))
// 	}
// }
// }

func scalaExportSymbols(file *index.ScalaFileSpec, resolvers []NameResolver) (exports []string, err error) {
	for _, names := range file.Extends {
	loop:
		for _, name := range names {
			// log.Println("resolving name %q in file %s", name, file.Filename)
			for _, resolver := range resolvers {
				if fqn, ok := resolver(name); ok {
					exports = append(exports, fqn)
					continue loop
				}
			}
			return nil, fmt.Errorf("failed to resolve name %q in file %s!", name, file.Filename)
		}
	}

	return
}

func resolveNameInLabelImportMap(resolved labelImportMap) NameResolver {
	in := make(map[string][]label.Label)
	for from, imports := range resolved {
		for imp := range imports {
			in[imp] = append(in[imp], from)
		}
	}
	return func(name string) (string, bool) {
		for imp := range in {
			if strings.HasSuffix(imp, "."+name) {
				return imp, true
			}
		}
		return "", false
	}
}

func resolveNameInFile(file *index.ScalaFileSpec) NameResolver {
	return func(name string) (string, bool) {
		suffix := "." + name
		for _, sym := range file.Traits {
			if strings.HasSuffix(sym, suffix) {
				return sym, true
			}
		}
		for _, sym := range file.Objects {
			if strings.HasSuffix(sym, suffix) {
				return sym, true
			}
		}
		for _, sym := range file.Classes {
			if strings.HasSuffix(sym, suffix) {
				return sym, true
			}
		}
		for _, sym := range file.Types {
			if strings.HasSuffix(sym, suffix) {
				return sym, true
			}
		}
		return "", false
	}
}

func shouldExcludeDep(c *config.Config, from label.Label) bool {
	return from.Name == "tests"
}

func makeLabeledListExpr(c *config.Config, kind string, from label.Label, resolved labelImportMap) build.Expr {
	dbg := debug
	if from.Repo == "" {
		from.Repo = c.RepoName
	}

	sc := getScalaConfig(c)

	list := make([]build.Expr, 0, len(resolved))
	seen := make(map[label.Label]bool)
	seen[from] = true

	id := 1
	for dep, imports := range resolved {
		if dbg {
			log.Printf("Writing %v dep %d: %v", from, id, dep)
		}

		if dep.Repo == "" {
			dep.Repo = c.RepoName
		}
		if seen[dep] {
			continue
		}
		if dep == label.NoLabel || dep == PlatformLabel || dep == from || from.Equal(dep) || isSameImport(sc, kind, from, dep) {
			continue
		}
		if shouldExcludeDep(c, dep) {
			continue
		}

		// relativize the depenency label.  For self-imports, this transforms into the empty label.
		dep = dep.Rel(from.Repo, from.Pkg)
		if dep == label.NoLabel {
			continue
		}

		str := &build.StringExpr{Value: dep.String()}
		list = append(list, str)
		seen[dep] = true

		if sc.explainDependencies {
			reasons := make([]string, 0, len(imports))
			for imp, origin := range imports {
				reasons = append(reasons, imp+" ("+origin.String()+")")
			}
			if len(reasons) == 0 {
				reasons = append(reasons, fmt.Sprintf("<unknown origin of %v>", importMapKeys(imports)))
			}
			sort.Strings(reasons)
			for _, reason := range reasons {
				str.Comments.Before = append(str.Comments.Before, build.Comment{Token: "# " + reason})
			}
		}

		// str.Comments.Suffix = []build.Comment{{Token: fmt.Sprintf("# %d", id)}}
		id += 1
	}

	sort.Slice(list, func(i, j int) bool {
		a := list[i].(*build.StringExpr)
		b := list[j].(*build.StringExpr)
		return a.Token < b.Token
	})

	return &build.ListExpr{List: list}
}

// getAttrFiles returns a list of source files for the 'srcs' attribute.  Each
// value is a repo-relative path.
func getAttrFiles(pkg ScalaPackage, r *rule.Rule, attrName string) (srcs []string, err error) {
	switch t := r.Attr(attrName).(type) {
	case *build.ListExpr:
		// example: ["foo.scala", "bar.scala"]
		for _, item := range t.List {
			switch elem := item.(type) {
			case *build.StringExpr:
				srcs = append(srcs, elem.Value)
			}
		}
	case *build.CallExpr:
		// example: glob(["**/*.scala"])
		if ident, ok := t.X.(*build.Ident); ok {
			switch ident.Name {
			case "glob":
				glob := parseGlob(pkg.File(), t)
				dir := filepath.Join(pkg.Dir(), pkg.Rel())
				srcs = append(srcs, applyGlob(glob, os.DirFS(dir))...)
			default:
				err = fmt.Errorf("not attempting to resolve function call %v(): consider making this simpler", ident.Name)
			}
		} else {
			err = fmt.Errorf("not attempting to resolve call expression %+v: consider making this simpler", t)
		}
	case *build.Ident:
		// example: srcs = LIST_OF_SOURCES
		srcs, err = globalStringList(pkg.File(), t)
		if err != nil {
			err = fmt.Errorf("faile to resolve resolve identifier %q (consider inlining it): %w", t.Name, err)
		}
	case nil:
		// TODO(pcj): should this be considered an error, or normal condition?
		// err = fmt.Errorf("rule has no 'srcs' attribute")
	default:
		err = fmt.Errorf("uninterpretable 'srcs' attribute type: %T", t)
	}

	return
}

func resolveScalaSrcs(dir string, from label.Label, kind string, srcs []string, parser ScalaFileParser) ([]*index.ScalaFileSpec, error) {
	if spec, err := parser.ParseScalaFiles(dir, from, kind, srcs...); err != nil {
		return nil, err
	} else {
		return spec.Srcs, nil
	}
}

// isUnqualifiedImport examples: 'CastDepthUtils._' or 'CastDepthUtils'.
func isUnqualifiedImport(imp string) bool {
	imp = strings.TrimSuffix(imp, "._")
	return strings.LastIndex(imp, ".") == -1
}
