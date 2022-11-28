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
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// a lazily-computed list of resolvers that implement LabelOwner
var labelOwners []crossresolve.LabelOwner

func getLabelOwners() []crossresolve.LabelOwner {
	if labelOwners == nil {
		for _, resolver := range crossresolve.Resolvers().ByName() {
			if labelOwner, ok := resolver.(crossresolve.LabelOwner); ok {
				labelOwners = append(labelOwners, labelOwner)
			}
		}
	}
	return labelOwners
}

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
	from := label.New("", pkg.Rel(), r.Name())
	files := make([]*index.ScalaFileSpec, 0)

	srcs, err := getAttrFiles(pkg, r, "srcs")
	if err != nil {
		log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
		return nil
	}

	if len(srcs) > 0 {
		// log.Printf("skipping %s //%s:%s (no srcs)", r.Kind(), pkg.Rel(), r.Name())
		// return nil
		files, err = resolveScalaSrcs(pkg.Dir(), from, r.Kind(), srcs, pkg.ScalaFileParser())
		if err != nil {
			log.Printf("skipping %s //%s:%s (%v)", r.Kind(), pkg.Rel(), r.Name(), err)
			return nil
		}

	}

	r.SetPrivateAttr(config.GazelleImportsKey, files)
	r.SetPrivateAttr(resolverImpLangPrivateKey, "java")
	// r.SetPrivateAttr(resolverImpLangPrivateKey, ScalaLangName)

	return &scalaExistingRuleRule{cfg, pkg, r, files, s.isBinaryRule}
}

// scalaExistingRuleRule implements RuleProvider for existing scala rules.
type scalaExistingRuleRule struct {
	cfg          *RuleConfig
	pkg          ScalaPackage
	rule         *rule.Rule
	files        []*index.ScalaFileSpec
	isBinaryRule bool
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
	lang := ScalaLangName
	if _, ok := sc.mapKindImportNames[r.Kind()]; ok {
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
func (s *scalaExistingRuleRule) Resolve(c *config.Config, ix *resolve.RuleIndex, r *rule.Rule, importsRaw interface{}, from label.Label) {
	// dbg := debug

	files, ok := importsRaw.([]*index.ScalaFileSpec)
	if !ok {
		return
	}

	sc := getScalaConfig(c)

	importRegistry := s.pkg.ScalaImportRegistry()
	imports := make(ImportOriginMap)

	impLang := r.Kind()
	if overrideImpLang, ok := r.PrivateAttr(resolverImpLangPrivateKey).(string); ok {
		impLang = overrideImpLang
	}

	if debug {
		log.Println(from, "| BEGIN RESOLVE", impLang)
	}

	// --- Gather imports ---

	// direct
	for _, file := range files {
		for _, imp := range file.Imports {
			imports.Add(imp, NewDirectImportOrigin(file))
		}
	}

	// if this rule has a main_class
	if mainClass := r.AttrString("main_class"); mainClass != "" {
		imports.Add(mainClass, &ImportOrigin{Kind: ImportKindMainClass})
	}

	// gather implicit imports
	implicits := make(collections.StringStack, 0)
	for src := range imports {
		for _, dst := range sc.GetImplicitImports(impLang, src) {
			implicits.Push(dst)
			imports.Add(dst, NewImplicitImportOrigin(src))
		}
	}
	// gather transitive implicits
	for !implicits.IsEmpty() {
		src, _ := implicits.Pop()
		for _, dst := range sc.GetImplicitImports(impLang, src) {
			implicits.Push(dst)
			imports.Add(dst, NewImplicitImportOrigin(src))
		}
	}

	resolved := NewLabelImportMap()

	// resolve this (mostly direct) initial set
	resolveImports(c, ix, importRegistry, impLang, r.Kind(), from, imports, resolved)

	unresolved := resolved[label.NoLabel]
	if debug && len(unresolved) > 0 {
		// panic(fmt.Sprintf("%v has unresolved dependencies: %v", from, unresolved))
		log.Printf("%v has unresolved dependencies: %v", from, unresolved)
	}

	if len(resolved) > 0 {
		labelOwners := getLabelOwners()
		keep := func(expr build.Expr) bool {
			return shouldKeep(expr, labelOwners...)
		}
		depsExpr := makeLabeledListExpr(c, r.Kind(), keep, r.Attr("deps"), from, resolved)
		r.SetAttr("deps", depsExpr)

		if len(unresolved) > 0 && sc.explainDependencies {
			commentUnresolvedImports(unresolved, r, "srcs")
		}
	}

	if debug {
		log.Println(from, "| END RESOLVE", impLang)
		// printRules(r)
	}
}

func commentUnresolvedImports(unresolved ImportOriginMap, r *rule.Rule, attrName string) {
	srcs := r.Attr(attrName)
	if srcs == nil {
		return
	}
	srcs.Comment().Before = nil

	imports := make([]string, 0, len(unresolved))
	for imp := range unresolved {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	for _, imp := range imports {
		origin := unresolved[imp]
		log.Println(imp, origin)
		srcs.Comment().Before = append(srcs.Comment().Before, build.Comment{
			Token: fmt.Sprintf("# unresolved: %s (%s)", imp, origin.String()),
		})
	}
}

func resolveNameInLabelImportMap(resolved LabelImportMap) NameResolver {
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

func makeLabeledListExpr(c *config.Config, kind string, shouldKeep func(build.Expr) bool, existingDeps build.Expr, from label.Label, resolved LabelImportMap) build.Expr {
	dbg := false

	if from.Repo == "" {
		from.Repo = c.RepoName
	}

	sc := getScalaConfig(c)

	list := make([]build.Expr, 0, len(resolved))
	seen := make(map[label.Label]bool)
	seen[from] = true

	if deps, ok := existingDeps.(*build.ListExpr); ok {
		for _, expr := range deps.List {
			if shouldKeep(expr) {
				list = append(list, expr)
			}
		}
	}

	// make a mapping of final deps to be included.  Getting strange behavior by
	// just creating a build.ListExpr and sorting that list directly.
	keeps := make(map[string]ImportOriginMap)

	for dep, imports := range resolved {
		if dbg {
			log.Println("makeLabeledListExpr: processing: ", dep)
		}

		if dep.Repo == "" {
			dep.Repo = c.RepoName
		}
		if seen[dep] {
			if dbg {
				log.Println("makeLabeledListExpr: seen: ", dep)
			}
			continue
		}
		if dep == label.NoLabel || dep == PlatformLabel || dep == from || from.Equal(dep) || isSameImport(sc, kind, from, dep) {
			if dbg {
				log.Println("makeLabeledListExpr: none-or-self: ", dep)
			}
			continue
		}
		// if shouldExcludeDep(c, dep) {
		// 	if dbg {
		// 		log.Println("makeLabeledListExpr: should-exclude: ", dep)
		// 	}
		// 	continue
		// }

		// relativize the depenency label.  For self-imports, this transforms into the empty label.
		dep = dep.Rel(from.Repo, from.Pkg)
		if dep == label.NoLabel {
			if dbg {
				log.Println("makeLabeledListExpr: relative no-label: ", dep)
			}
			continue
		}

		keeps[dep.String()] = imports
		seen[dep] = true
	}

	keys := make([]string, 0, len(keeps))
	for key := range keeps {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for id, dep := range keys {
		imports := keeps[dep]
		str := &build.StringExpr{Value: dep}
		if sc.explainDependencies {
			explainDependencies(str, imports)
			if debug {
				str.Comments.Suffix = []build.Comment{{Token: fmt.Sprintf("# %d", id)}}
			}
		}
		list = append(list, str)
	}

	return &build.ListExpr{List: list}
}

func explainDependencies(str *build.StringExpr, imports ImportOriginMap) {
	reasons := make([]string, 0, len(imports))
	for imp, origin := range imports {
		reason := imp + " (" + origin.String() + ")"
		reasons = append(reasons, reason)
	}
	if len(reasons) == 0 {
		reasons = append(reasons, fmt.Sprintf("<unknown origin of %v>", imports.Keys()))
	}
	reasons = protoc.DeduplicateAndSort(reasons)
	for _, reason := range reasons {
		str.Comments.Before = append(str.Comments.Before, build.Comment{Token: "# " + reason})
	}
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

func shouldKeep(expr build.Expr, labelOwners ...crossresolve.LabelOwner) bool {
	// does it have a '# keep' directive?
	if rule.ShouldKeep(expr) {
		return true
	}

	// is the expression something we can parse as a label?
	// If not, just leave it be.
	from := scalaDepLabel(expr)
	if from == label.NoLabel {
		return true
	}

	// if we can find a resolver than manages/owns this label, remove it;
	// the resolver should cross-resolve the import again
	for _, resolver := range labelOwners {
		if resolver.IsLabelOwner(from, func(from label.Label) (*rule.Rule, bool) {
			return nil, false
		}) {
			return false
		}
	}

	// we didn't find an owner so keep it, it's not a managed dependency.
	return true
}

func printRules(rules ...*rule.Rule) {
	file := rule.EmptyFile("", "")
	for _, r := range rules {
		r.Insert(file)
	}
	fmt.Println(string(file.Format()))
}

// scalaDepLabel returns the label from an expression like
// "@maven//:guava" or scala_dep("@maven//:guava")
func scalaDepLabel(expr build.Expr) label.Label {
	switch t := expr.(type) {
	case *build.StringExpr:
		if from, err := label.Parse(t.Value); err != nil {
			return label.NoLabel
		} else {
			return from
		}
	case *build.CallExpr:
		if ident, ok := t.X.(*build.Ident); ok && ident.Name == "scala_dep" {
			if len(t.List) == 0 {
				return label.NoLabel
			}
			first := t.List[0]
			if str, ok := first.(*build.StringExpr); ok {
				if from, err := label.Parse(str.Value); err != nil {
					return label.NoLabel
				} else {
					return from
				}
			}
		}
	}

	return label.NoLabel
}
