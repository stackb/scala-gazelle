package scalaconfig

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bmatcuk/doublestar/v4"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

type debugAnnotation int

const (
	DebugUnknown  debugAnnotation = 0
	DebugImports  debugAnnotation = 1
	DebugExports  debugAnnotation = 2
	scalaLangName                 = "scala"
)

const (
	scalaDebugDirective             = "scala_debug"
	scalaFixWildcardImportDirective = "scala_fix_wildcard_imports"
	scalaRuleDirective              = "scala_rule"
	resolveGlobDirective            = "resolve_glob"
	resolveConflictsDirective       = "resolve_conflicts"
	scalaDepsCleanerDirective       = "scala_deps_cleaner"
	resolveWithDirective            = "resolve_with"
	resolveFileSymbolName           = "resolve_file_symbol_name"
	resolveKindRewriteNameDirective = "resolve_kind_rewrite_name"
)

func DirectiveNames() []string {
	return []string{
		scalaDebugDirective,
		scalaFixWildcardImportDirective,
		scalaRuleDirective,
		resolveGlobDirective,
		resolveConflictsDirective,
		scalaDepsCleanerDirective,
		resolveWithDirective,
		resolveFileSymbolName,
		resolveKindRewriteNameDirective,
	}
}

// Config represents the config extension for the a scala package.
type Config struct {
	config                 *config.Config
	rel                    string
	universe               resolver.Universe
	overrides              []*overrideSpec
	implicitImports        []*implicitImportSpec
	resolveFileSymbolNames []*resolveFileSymbolNameSpec
	fixWildcardImportSpecs []*fixWildcardImportSpec
	rules                  map[string]*scalarule.Config
	labelNameRewrites      map[string]resolver.LabelNameRewriteSpec
	annotations            map[debugAnnotation]interface{}
	conflictResolvers      []resolver.ConflictResolver
	depsCleaners           []resolver.DepsCleaner
}

// newScalaConfig initializes a new Config.
func New(universe resolver.Universe, config *config.Config, rel string) *Config {
	return &Config{
		config:            config,
		rel:               rel,
		universe:          universe,
		annotations:       make(map[debugAnnotation]interface{}),
		labelNameRewrites: make(map[string]resolver.LabelNameRewriteSpec),
		rules:             make(map[string]*scalarule.Config),
	}
}

// Get returns the scala config.  Can be nil.
func Get(config *config.Config) *Config {
	if existingExt, ok := config.Exts[scalaLangName]; ok {
		return existingExt.(*Config)
	} else {
		return nil
	}
}

// getOrCreateScalaConfig either inserts a new config into the map under the
// language name or replaces it with a clone.
func GetOrCreate(universe resolver.Universe, config *config.Config, rel string) *Config {
	var cfg *Config
	if existingExt, ok := config.Exts[scalaLangName]; ok {
		cfg = existingExt.(*Config).clone(config, rel)
	} else {
		cfg = New(universe, config, rel)
	}
	config.Exts[scalaLangName] = cfg
	return cfg
}

// clone copies this config to a new one.
func (c *Config) clone(config *config.Config, rel string) *Config {
	clone := New(c.universe, config, rel)
	for k, v := range c.annotations {
		clone.annotations[k] = v
	}
	for k, v := range c.rules {
		clone.rules[k] = v.Clone()
	}
	for k, v := range c.labelNameRewrites {
		clone.labelNameRewrites[k] = v
	}
	if c.overrides != nil {
		clone.overrides = c.overrides[:]
	}
	if c.implicitImports != nil {
		clone.implicitImports = c.implicitImports[:]
	}
	if c.conflictResolvers != nil {
		clone.conflictResolvers = c.conflictResolvers[:]
	}
	if c.depsCleaners != nil {
		clone.depsCleaners = c.depsCleaners[:]
	}
	if c.resolveFileSymbolNames != nil {
		clone.resolveFileSymbolNames = c.resolveFileSymbolNames[:]
	}
	if c.fixWildcardImportSpecs != nil {
		clone.fixWildcardImportSpecs = c.fixWildcardImportSpecs[:]
	}
	return clone
}

// Config returns the parent gazelle configuration
func (c *Config) Config() *config.Config {
	return c.config
}

// Rel returns the parent gazelle relative path
func (c *Config) Rel() string {
	return c.rel
}

func (c *Config) CanProvide(from label.Label) bool {
	for _, provider := range c.universe.SymbolProviders() {
		if provider.CanProvide(from, c.universe.GetKnownRule) {
			return true
		}
	}
	return false
}

func (c *Config) ResolveConflict(r *rule.Rule, imports resolver.ImportMap, imp *resolver.Import, symbol *resolver.Symbol) (*resolver.Symbol, bool) {
	for _, resolver := range c.conflictResolvers {
		if resolved, ok := resolver.ResolveConflict(c.universe, r, imports, imp, symbol); ok {
			return resolved, true
		}
	}
	return nil, false
}

// GetKnownRule translates relative labels into their absolute form.
func (c *Config) GetKnownRule(from label.Label) (*rule.Rule, bool) {
	if from.Name == "" {
		return nil, false
	}
	if from.Repo == "" && from.Pkg == "" {
		from = label.Label{Pkg: c.rel, Name: from.Name}
	}
	return c.universe.GetKnownRule(from)
}

// parseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (c *Config) ParseDirectives(directives []rule.Directive) (err error) {
	for _, d := range directives {
		switch d.Key {
		case scalaRuleDirective:
			err = c.parseScalaRuleDirective(d)
			if err != nil {
				return fmt.Errorf(`invalid directive: "gazelle:%s %s": %w`, d.Key, d.Value, err)
			}
		case scalaFixWildcardImportDirective:
			c.parseFixWildcardImport(d)
		case resolveGlobDirective:
			c.parseResolveGlobDirective(d)
		case resolveWithDirective:
			c.parseResolveWithDirective(d)
		case resolveFileSymbolName:
			c.parseResolveFileSymbolNames(d)
		case resolveKindRewriteNameDirective:
			c.parseResolveKindRewriteNameDirective(d)
		case resolveConflictsDirective:
			if err := c.parseResolveConflictsDirective(d); err != nil {
				return err
			}
		case scalaDepsCleanerDirective:
			if err := c.parseScalaDepsCleanerDirective(d); err != nil {
				return err
			}
		case scalaDebugDirective:
			if err := c.parseScalaAnnotation(d); err != nil {
				return err
			}
		}
	}
	return
}

func (c *Config) parseScalaRuleDirective(d rule.Directive) error {
	fields := strings.Fields(d.Value)
	if len(fields) < 3 {
		return fmt.Errorf("expected three or more fields, got %d", len(fields))
	}
	name, param, value := fields[0], fields[1], strings.Join(fields[2:], " ")
	r, err := c.getOrCreateScalaRuleConfig(name)
	if err != nil {
		return err
	}
	return r.ParseDirective(name, param, value)
}

func (c *Config) parseResolveGlobDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	o := overrideSpec{}
	var lbl string
	if len(parts) != 4 {
		return
	}
	if parts[0] != scalaLangName {
		return
	}

	o.imp.Lang = parts[0]
	o.lang = parts[1]
	o.imp.Imp = parts[2]
	lbl = parts[3]

	var err error
	o.dep, err = label.Parse(lbl)
	if err != nil {
		log.Fatalf("bad gazelle:%s directive value %q: %v", resolveGlobDirective, d.Value, err)
		return
	}
	c.overrides = append(c.overrides, &o)
}

func (c *Config) parseResolveWithDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) < 3 {
		log.Printf("invalid gazelle:%s directive: expected 3+ parts, got %d (%v)", resolveWithDirective, len(parts), parts)
		return
	}
	c.implicitImports = append(c.implicitImports, &implicitImportSpec{
		lang: parts[0],
		imp:  parts[1],
		deps: parts[2:],
	})
}

func (c *Config) parseFixWildcardImport(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) < 2 {
		log.Fatalf("invalid gazelle:%s directive: expected [FILENAME_PATTERN [+|-]IMPORT_PATTERN...], got %v", scalaFixWildcardImportDirective, parts)
		return
	}
	filenamePattern := parts[0]

	for _, part := range parts[1:] {
		intent := collections.ParseIntent(part)
		c.fixWildcardImportSpecs = append(c.fixWildcardImportSpecs, &fixWildcardImportSpec{
			filenamePattern: filenamePattern,
			importPattern:   *intent,
		})
	}

}

func (c *Config) parseResolveFileSymbolNames(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) < 2 {
		log.Printf("invalid gazelle:%s directive: expected [FILENAME_PATTERN [+|-]SYMBOLS...], got %v", resolveFileSymbolName, parts)
		return
	}
	pattern := parts[0]

	for _, part := range parts[1:] {
		intent := collections.ParseIntent(part)
		c.resolveFileSymbolNames = append(c.resolveFileSymbolNames, &resolveFileSymbolNameSpec{
			pattern:    pattern,
			symbolName: *intent,
		})
	}
}

func (c *Config) parseResolveKindRewriteNameDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) != 3 {
		log.Printf("invalid gazelle:%s directive: expected [KIND SRC_NAME DST_NAME], got %v", resolveKindRewriteNameDirective, parts)
		return
	}
	kind := parts[0]
	src := parts[1]
	dst := parts[2]

	c.labelNameRewrites[kind] = resolver.LabelNameRewriteSpec{Src: src, Dst: dst}
}

func (c *Config) parseResolveConflictsDirective(d rule.Directive) error {
	for _, key := range strings.Fields(d.Value) {
		intent := collections.ParseIntent(key)
		if intent.Want {
			resolver, ok := c.universe.GetConflictResolver(intent.Value)
			if !ok {
				return fmt.Errorf("invalid directive gazelle:%s: unknown conflict resolver %q", d.Key, intent.Value)
			}
			for _, cr := range c.conflictResolvers {
				if cr.Name() == intent.Value {
					break
				}
			}
			c.conflictResolvers = append(c.conflictResolvers, resolver)
		} else {
			for i, cr := range c.conflictResolvers {
				if cr.Name() == intent.Value {
					c.conflictResolvers = removeConflictResolver(c.conflictResolvers, i)
				}
			}
		}
	}
	return nil
}

func (c *Config) parseScalaDepsCleanerDirective(d rule.Directive) error {
	for _, key := range strings.Fields(d.Value) {
		intent := collections.ParseIntent(key)
		if intent.Want {
			resolver, ok := c.universe.GetDepsCleaner(intent.Value)
			if !ok {
				return fmt.Errorf("invalid directive gazelle:%s: unknown scala deps cleaner %q", d.Key, intent.Value)
			}
			for _, cr := range c.depsCleaners {
				if cr.Name() == intent.Value {
					break
				}
			}
			c.depsCleaners = append(c.depsCleaners, resolver)
		} else {
			for i, cr := range c.depsCleaners {
				if cr.Name() == intent.Value {
					c.depsCleaners = removeDepsCleaner(c.depsCleaners, i)
				}
			}
		}
	}
	return nil
}

func (c *Config) parseScalaAnnotation(d rule.Directive) error {
	for _, key := range strings.Fields(d.Value) {
		intent := collections.ParseIntent(key)
		annot := parseAnnotation(intent.Value)
		if annot == DebugUnknown {
			return fmt.Errorf("invalid directive gazelle:%s: unknown annotation value '%v'", d.Key, intent.Value)
		}
		if intent.Want {
			var val interface{}
			c.annotations[annot] = val
		} else {
			delete(c.annotations, annot)
		}
	}
	return nil
}

func (c *Config) getOrCreateScalaRuleConfig(name string) (*scalarule.Config, error) {
	r, ok := c.rules[name]
	if !ok {
		r = scalarule.NewConfig(c.config, name)
		r.Implementation = name
		c.rules[name] = r
	}
	return r, nil
}

func (c *Config) GetImplicitImports(lang, imp string) (deps []string) {
	for _, d := range c.implicitImports {
		if d.lang != lang {
			continue
		}
		if d.imp != imp {
			continue
		}
		deps = append(deps, d.deps...)
	}
	return
}

// ConfiguredRules returns an ordered list of configured rules
func (c *Config) ConfiguredRules() []*scalarule.Config {

	names := make([]string, 0)
	for name := range c.rules {
		names = append(names, name)
	}
	sort.Strings(names)
	rules := make([]*scalarule.Config, len(names))
	for i, name := range names {
		rules[i] = c.rules[name]
	}
	return rules
}

func (c *Config) ShouldAnnotateImports() bool {
	_, ok := c.annotations[DebugImports]
	return ok
}

func (c *Config) ShouldAnnotateExports() bool {
	_, ok := c.annotations[DebugExports]
	return ok
}

// ShouldFixWildcardImport tests whether the given symbol name pattern
// should be resolved within the scope of the given filename pattern.
// resolveFileSymbolNameSpecs represent a whitelist; if no patterns match, false
// is returned.
func (c *Config) ShouldFixWildcardImport(filename, wimp string) bool {
	for _, spec := range c.fixWildcardImportSpecs {
		hasStarChar := strings.Contains(spec.filenamePattern, "*")
		if hasStarChar {
			if ok, _ := doublestar.Match(spec.filenamePattern, filename); !ok {
				// log.Println("should fix wildcard import? FILENAME GLOB MATCH FAILED", filename, spec.filenamePattern)
				continue
			}
		} else {
			if !strings.HasSuffix(filename, spec.filenamePattern) {
				// log.Println("should fix wildcard import? FILENAME SUFFIX MATCH FAILED", filename, spec.filenamePattern)
				continue
			}
		}
		if ok, _ := doublestar.Match(spec.importPattern.Value, wimp); !ok {
			// log.Println("should fix wildcard import? IMPORT PATTERN MATCH FAILED", filename, spec.importPattern.Value, wimp)
			continue
		}
		return spec.importPattern.Want
	}
	return false
}

// ShouldResolveFileSymbolName tests whether the given symbol name pattern
// should be resolved within the scope of the given filename pattern.
// resolveFileSymbolNameSpecs represent a whitelist; if no patterns match, false
// is returned.
func (c *Config) ShouldResolveFileSymbolName(filename, name string) bool {
	for _, spec := range c.resolveFileSymbolNames {
		if ok, _ := doublestar.Match(spec.pattern, filename); !ok {
			continue
		}
		if ok, _ := doublestar.Match(spec.symbolName.Value, name); !ok {
			continue
		}
		return spec.symbolName.Want
	}
	return false
}

func (c *Config) Comment() build.Comment {
	return build.Comment{Token: "# " + c.String()}
}

func (c *Config) String() string {
	return fmt.Sprintf("Config rel=%q, annotations=%+v", c.rel, c.annotations)
}

// MaybeRewrite takes a rule kind and a from label and possibly transforms the
// label name based on the configuration of label name rewrites. For example,
// consider a  rule macro `my_scala_app` having a label name ':app', and a file
// Helper.scala, and the definition of the macro passes `srcs` to an an internal
// scala_library named ':lib' (that includes Helper.scala).  For other scala
// rules that import 'Helper', we want to depend on `//somepkg:lib` rather than
// `somepkg:app`.
func (c *Config) MaybeRewrite(kind string, from label.Label) label.Label {
	if spec, ok := c.labelNameRewrites[kind]; ok {
		return spec.Rewrite(from)
	}
	return from
}

// mergeListExpr takes the given list of existing deps and a list of dependency
// labels and merges it into a final sorted list.
func mergeListExpr(target *build.ListExpr, deps map[label.Label]bool) {
	for dep, want := range deps {
		if !want {
			continue
		}
		target.List = append(target.List, &build.StringExpr{Value: dep.String()})
	}

	sort.Slice(target.List, func(i, j int) bool {
		a, aIsString := target.List[i].(*build.StringExpr)
		b, bIsString := target.List[j].(*build.StringExpr)
		if aIsString && bIsString {
			return a.Token < b.Token
		}
		return false
	})
}

func (c *Config) Imports(imports resolver.ImportMap, r *rule.Rule, attrName string, from label.Label) {
	c.ruleAttrMergeDeps(imports, r, c.shouldKeepDep, c.ShouldAnnotateImports(), attrName, from)
}

func (c *Config) Exports(exports resolver.ImportMap, r *rule.Rule, attrName string, from label.Label) {
	c.ruleAttrMergeDeps(exports, r, shouldKeepExport, c.ShouldAnnotateExports(), attrName, from)
}

func (c *Config) ruleAttrMergeDeps(
	imports resolver.ImportMap,
	r *rule.Rule,
	shouldKeep func(build.Expr) bool,
	shouldAnnotateSrcs bool,
	attrName string,
	from label.Label,
) {
	labels := imports.Deps(c.MaybeRewrite(r.Kind(), from))
	deps := make(map[label.Label]bool)
	for _, l := range labels {
		deps[l] = true
	}

	// for _, impl := range c.depsCleaners {
	// 	impl.CleanDeps(deps, r, from)
	// }

	next := cleanDepsList(r.Attr(attrName), deps, shouldKeep, attrName, from)

	mergeListExpr(next, deps)
	if len(next.List) > 0 {
		r.SetAttr(attrName, next)
	} else {
		r.DelAttr(attrName)
	}

	if shouldAnnotateSrcs {
		comments := r.AttrComments("srcs")
		if comments != nil {
			prefix := attrName + ": "
			annotateImports(imports, comments, prefix)
		}
	}
}

// cleanDepsList takes the AST state of a (deps or exports) attr, processes each
// expr in the list
func cleanDepsList(attrValue build.Expr, deps map[label.Label]bool, keep func(expr build.Expr) bool, attrName string, from label.Label) *build.ListExpr {
	next := new(build.ListExpr)
	if attrValue == nil {
		return next
	}
	if listExpr, ok := attrValue.(*build.ListExpr); ok {
		for _, expr := range listExpr.List {
			if keep(expr) {
				dep := labelFromDepExpr(expr)
				if rule.ShouldKeep(expr) && deps[dep] {
					log.Printf(`%v: in attr %q, "%v" does not need a '# keep' directive (fixed)`, attrName, from, dep)
					continue
				}
				next.List = append(next.List, expr)
			}
		}
	}
	return next
}

func shouldKeepExport(expr build.Expr) bool {
	// does it have a '# keep' directive?
	if rule.ShouldKeep(expr) {
		return true
	}

	// is the expression something we can parse as a label? If not, just leave
	// it be.
	from := labelFromDepExpr(expr)

	if from == label.NoLabel {
		return true
	}

	// delete exports by default, expect caller to have 'keep' comments
	return false
}

func (c *Config) shouldKeepDep(expr build.Expr) bool {
	// does it have a '# keep' directive?
	if rule.ShouldKeep(expr) {
		return true
	}

	// is the expression something we can parse as a label? If not, just leave
	// it be.
	from := labelFromDepExpr(expr)
	if from == label.NoLabel {
		return true
	}

	// if we can find a provider for this label, remove it (it should have been
	// resolved again if still wanted)
	if c.CanProvide(from) {
		return false
	}

	// we didn't find an owner so keep just it, it's not a managed dependency.
	return true
}

// labelFromDepExpr returns the label from an expression like "@maven//:guava"
// or scala_dep("@maven//:guava")
func labelFromDepExpr(expr build.Expr) label.Label {
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

type overrideSpec struct {
	imp  resolve.ImportSpec
	lang string
	dep  label.Label
}

type implicitImportSpec struct {
	// lang is the language to which this implicit applies.  Always 'scala' for now.
	lang string
	// imp is the "source" dependency (e.g. LazyLogging)
	imp string
	// dep is the "destination" dependencies (e.g. org.slf4j.Logger)
	deps []string
}

type resolveFileSymbolNameSpec struct {
	// pattern is the filename glob pattern to test
	pattern string
	// symbol is the symbol name to resolve
	symbolName collections.Intent
}

type fixWildcardImportSpec struct {
	// filenamePattern is the filename glob wildcard filenamePattern to test
	filenamePattern string
	importPattern   collections.Intent
}

func parseAnnotation(val string) debugAnnotation {
	switch val {
	case "imports":
		return DebugImports
	case "exports":
		return DebugExports
	default:
		return DebugUnknown
	}
}

func removeConflictResolver(slice []resolver.ConflictResolver, index int) []resolver.ConflictResolver {
	return append(slice[:index], slice[index+1:]...)
}

func removeDepsCleaner(slice []resolver.DepsCleaner, index int) []resolver.DepsCleaner {
	return append(slice[:index], slice[index+1:]...)
}

func annotateImports(imports resolver.ImportMap, comments *build.Comments, prefix string) {
	comments.Before = nil
	for _, key := range imports.Keys() {
		imp := imports[key]
		comment := setCommentPrefix(imp.Comment(), prefix)
		comments.Before = append(comments.Before, comment)
	}
}

func setCommentPrefix(comment build.Comment, prefix string) build.Comment {
	comment.Token = "# " + prefix + strings.TrimSpace(strings.TrimPrefix(comment.Token, "#"))
	return comment
}
