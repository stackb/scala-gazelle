package scala

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

type annotation int

const (
	AnnotateUnknown annotation = 0
	AnnotateImports annotation = 1
	AnnotateExports annotation = 2
)

const (
	scalaAnnotateDirective          = "scala_annotate"
	scalaRuleDirective              = "scala_rule"
	resolveGlobDirective            = "resolve_glob"
	resolveConflictsDirective       = "resolve_conflicts"
	resolveWithDirective            = "resolve_with"
	resolveFileSymbolName           = "resolve_file_symbol_name"
	resolveKindRewriteNameDirective = "resolve_kind_rewrite_name"
)

// scalaConfig represents the config extension for the a scala package.
type scalaConfig struct {
	config                 *config.Config
	rel                    string
	universe               resolver.Universe
	overrides              []*overrideSpec
	implicitImports        []*implicitImportSpec
	resolveFileSymbolNames []*resolveFileSymbolNameSpec
	rules                  map[string]*scalarule.Config
	labelNameRewrites      map[string]resolver.LabelNameRewriteSpec
	annotations            map[annotation]interface{}
	conflictResolvers      []resolver.ConflictResolver
}

// newScalaConfig initializes a new scalaConfig.
func newScalaConfig(universe resolver.Universe, config *config.Config, rel string) *scalaConfig {
	return &scalaConfig{
		config:            config,
		rel:               rel,
		universe:          universe,
		annotations:       make(map[annotation]interface{}),
		labelNameRewrites: make(map[string]resolver.LabelNameRewriteSpec),
		rules:             make(map[string]*scalarule.Config),
	}
}

// getScalaConfig returns the scala config.  Can be nil.
func getScalaConfig(config *config.Config) *scalaConfig {
	if existingExt, ok := config.Exts[scalaLangName]; ok {
		return existingExt.(*scalaConfig)
	} else {
		return nil
	}
}

// getOrCreateScalaConfig either inserts a new config into the map under the
// language name or replaces it with a clone.
func getOrCreateScalaConfig(universe resolver.Universe, config *config.Config, rel string) *scalaConfig {
	var cfg *scalaConfig
	if existingExt, ok := config.Exts[scalaLangName]; ok {
		cfg = existingExt.(*scalaConfig).clone(config, rel)
	} else {
		cfg = newScalaConfig(universe, config, rel)
	}
	config.Exts[scalaLangName] = cfg
	return cfg
}

// clone copies this config to a new one.
func (c *scalaConfig) clone(config *config.Config, rel string) *scalaConfig {
	clone := newScalaConfig(c.universe, config, rel)
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
	if c.resolveFileSymbolNames != nil {
		clone.resolveFileSymbolNames = c.resolveFileSymbolNames[:]
	}
	return clone
}

func (c *scalaConfig) canProvide(from label.Label) bool {
	for _, provider := range c.universe.SymbolProviders() {
		if provider.CanProvide(from, c.universe.GetKnownRule) {
			return true
		}
	}
	return false
}

func (c *scalaConfig) resolveConflict(r *rule.Rule, imports resolver.ImportMap, imp *resolver.Import, symbol *resolver.Symbol) (*resolver.Symbol, bool) {
	for _, resolver := range c.conflictResolvers {
		if resolved, ok := resolver.ResolveConflict(c.universe, r, imports, imp, symbol); ok {
			return resolved, true
		}
	}
	return nil, false
}

// GetKnownRule translates relative labels into their absolute form.
func (c *scalaConfig) GetKnownRule(from label.Label) (*rule.Rule, bool) {
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
func (c *scalaConfig) parseDirectives(directives []rule.Directive) (err error) {
	for _, d := range directives {
		switch d.Key {
		case scalaRuleDirective:
			err = c.parseScalaRuleDirective(d)
			if err != nil {
				return fmt.Errorf(`invalid directive: "gazelle:%s %s": %w`, d.Key, d.Value, err)
			}
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
		case scalaAnnotateDirective:
			if err := c.parseScalaAnnotation(d); err != nil {
				return err
			}
		}
	}
	return
}

func (c *scalaConfig) parseScalaRuleDirective(d rule.Directive) error {
	fields := strings.Fields(d.Value)
	if len(fields) < 3 {
		return fmt.Errorf("expected three or more fields, got %d", len(fields))
	}
	name, param, value := fields[0], fields[1], strings.Join(fields[2:], " ")
	r, err := c.getOrCreateScalaRuleConfig(c.config, name)
	if err != nil {
		return err
	}
	return r.ParseDirective(name, param, value)
}

func (c *scalaConfig) parseResolveGlobDirective(d rule.Directive) {
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

func (c *scalaConfig) parseResolveWithDirective(d rule.Directive) {
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

func (c *scalaConfig) parseResolveFileSymbolNames(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) < 2 {
		log.Printf("invalid gazelle:%s directive: expected [FILENAME_PATTERN [+|-]SYMBOLS...], got %v", resolveKindRewriteNameDirective, parts)
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

func (c *scalaConfig) parseResolveKindRewriteNameDirective(d rule.Directive) {
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

func (c *scalaConfig) parseResolveConflictsDirective(d rule.Directive) error {
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

func (c *scalaConfig) parseScalaAnnotation(d rule.Directive) error {
	for _, key := range strings.Fields(d.Value) {
		intent := collections.ParseIntent(key)
		annot := parseAnnotation(intent.Value)
		if annot == AnnotateUnknown {
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

func (c *scalaConfig) getOrCreateScalaRuleConfig(config *config.Config, name string) (*scalarule.Config, error) {
	r, ok := c.rules[name]
	if !ok {
		r = scalarule.NewConfig(config, name)
		r.Implementation = name
		c.rules[name] = r
	}
	return r, nil
}

func (c *scalaConfig) getImplicitImports(lang, imp string) (deps []string) {
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

// configuredRules returns an ordered list of configured rules
func (c *scalaConfig) configuredRules() []*scalarule.Config {

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

func (c *scalaConfig) shouldAnnotateImports() bool {
	_, ok := c.annotations[AnnotateImports]
	return ok
}

func (c *scalaConfig) shouldAnnotateExports() bool {
	_, ok := c.annotations[AnnotateExports]
	return ok
}

// ShouldResolveFileSymbolName tests whether the given symbol name pattern
// should be resolved within the scope of the given filename pattern.
// resolveFileSymbolNameSpecs represent a whitelist; if no patterns match, false
// is returned.
func (c *scalaConfig) ShouldResolveFileSymbolName(filename, name string) bool {
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

func (c *scalaConfig) Comment() build.Comment {
	return build.Comment{Token: "# " + c.String()}
}

func (c *scalaConfig) String() string {
	return fmt.Sprintf("scalaConfig rel=%q, annotations=%+v", c.rel, c.annotations)
}

func (c *scalaConfig) maybeRewrite(kind string, from label.Label) label.Label {
	if spec, ok := c.labelNameRewrites[kind]; ok {
		return spec.Rewrite(from)
	}
	return from
}

// mergeDeps takes the given list of existing deps and a list of dependency
// labels and merges it into a final list.
func mergeDeps(kind string, target *build.ListExpr, deps []label.Label) {
	for _, dep := range deps {
		str := &build.StringExpr{Value: dep.String()}
		target.List = append(target.List, str)
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

// cleanExports takes the given list of exports and removes those that are expected to
// be provided again.
func (c *scalaConfig) cleanExports(from label.Label, current build.Expr, newExports []label.Label) *build.ListExpr {
	incoming := make(map[label.Label]bool)
	for _, l := range newExports {
		incoming[l] = true
	}

	exports := &build.ListExpr{}
	if current != nil {
		if listExpr, ok := current.(*build.ListExpr); ok {
			for _, expr := range listExpr.List {
				if c.shouldKeepExport(expr) {
					dep := labelFromDepExpr(expr)
					if rule.ShouldKeep(expr) && incoming[dep] {
						log.Printf(`%v: in attr 'exports', "%v" does not need a '# keep' directive (fixed)`, from, dep)
						continue
					}
					exports.List = append(exports.List, expr)
				}
			}
		}
	}
	return exports
}

// cleanDeps takes the given list of deps and removes those that are expected to
// be provided again.
func (c *scalaConfig) cleanDeps(from label.Label, current build.Expr, newImports []label.Label) *build.ListExpr {
	incoming := make(map[label.Label]bool)
	for _, l := range newImports {
		incoming[l] = true
	}

	deps := &build.ListExpr{}
	if current != nil {
		if listExpr, ok := current.(*build.ListExpr); ok {
			for _, expr := range listExpr.List {
				if c.shouldKeepDep(expr) {
					dep := labelFromDepExpr(expr)
					if rule.ShouldKeep(expr) && incoming[dep] {
						log.Printf(`%v: in attr 'deps', "%v" does not need a '# keep' directive (fixed)`, from, dep)
						continue
					}
					deps.List = append(deps.List, expr)
				}
			}
		}
	}
	return deps
}

func (c *scalaConfig) shouldKeepExport(expr build.Expr) bool {
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

func (c *scalaConfig) shouldKeepDep(expr build.Expr) bool {
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
	if c.canProvide(from) {
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

func parseAnnotation(val string) annotation {
	switch val {
	case "imports":
		return AnnotateImports
	case "exports":
		return AnnotateExports
	default:
		return AnnotateUnknown
	}
}

func removeConflictResolver(slice []resolver.ConflictResolver, index int) []resolver.ConflictResolver {
	return append(slice[:index], slice[index+1:]...)
}
