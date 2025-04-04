package scalaconfig

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

type debugAnnotation int

const scalaLangName = "scala"

const (
	DebugUnknown        debugAnnotation = 0
	DebugImports        debugAnnotation = 1
	DebugExports        debugAnnotation = 2
	DebugDeps           debugAnnotation = 3
	DebugRule           debugAnnotation = 4
	DebugDepLabelOrigin debugAnnotation = 5
)

const (
	// Enable additional debugging
	//
	// gazelle:scala_debug true
	scalaDebugDirective = "scala_debug"

	// Change the logging level
	//
	// gazelle:scala_log_level WARN|INFO|DEBUG|TRACE
	scalaLogLevelDirective = "scala_log_level"

	// Limits scala-gazelle to only consider existing build files.  If false,
	// scala-gazelle itself will never generate new build files.  Defaults to
	// false.
	//
	// gazelle:scala_build_file_generate true
	scalaGenerateBuildFilesDirective = "scala_generate_build_files"

	// Turn on the wildcard import fixer
	//
	// gazelle:scala_fix_wildcard_imports .scala examples.aeron.api.proto._
	scalaFixWildcardImportDirective = "scala_fix_wildcard_imports"

	// Configure a scala rule
	//
	// gazelle:scala_rule RULE_NAME ATTRIBUTE VALUE
	// gazelle:scala_rule scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary
	scalaRuleDirective = "scala_rule"

	// Use glob for resolve overrides
	//
	// TODO(pcj): either remove this or implement it fully.  The directive
	// currently does nothing.
	resolveGlobDirective = "resolve_glob"

	// Turn on a conflict resolver
	//
	// # gazelle:resolve_conflicts predefined_label
	resolveConflictsDirective = "resolve_conflicts"

	// Turn on a deps cleaner
	//
	// # gazelle:scala_deps_cleaner scala_proto_grpc_deps_cleaner
	scalaDepsCleanerDirective = "scala_deps_cleaner"

	// Declare an implicit import link.  If B "resolves with" A and A is a dependency, also include B.
	//
	// # gazelle:resolve_with scala akka.actor.ActorSystem com.typesafe.config.Config
	resolveWithDirective = "resolve_with"

	// The 'gazelle:resolve_file_symbol_name' directive reads as follows "for
	// all files matching the glob '*.scala' (e.g., all scala files), enable
	// "name" resolution for names matching the glob '*' (basically, any name).
	//
	// So what is a 'name'?  When scala-gazelle parses a scala file, it produces
	// a proto message that lists all the imports statement in the file, all the
	// classes, traits, etc that are defined by the file.  It also has an array
	// (called "Names") that contains a list of raw strings containing basically
	// any capitalized symbol name found by traversing the AST.  For example, it
	// may be an array like ["Boolean", "Unit", "Map", "Option", "ToActorId",
	// ...]
	//
	// // Normally we only use the import statements to determine the required
	// deps for a scala_library rule.  However, the 'omnistac.gum.entity'
	// package is somewhat unique in that each scala file has its own rule, yet
	// all the files share the same scala package name.  So, while `Ids.scala`
	// defines `omnistac.gum.entity.ToActorId`, `User.scala` just uses the
	// symbol `ToActorId` without any corresponding import.  Yet, `:user_scala`
	// needs `:ids_scala` in its deps.  So how do we determine that?
	//
	// // This is what the resolve_file_symbol_name strategy does.  It permits
	// the use of the 'Names' metadata to try and find the corresponding
	// fully-qualified symbol name.  For example, when applying this strategy
	// for `User.scala`, we do a scope lookup of `ToActorId`, which in this case
	// completes to `omnistac.gum.entity.ToActorId`.  This fully-qualified
	// symbol is added to the list of required imports for the file, which is
	// then used to figure out the correct bazel dependency (':ids_scala').
	//
	// # gazelle:resolve_file_symbol_name *.scala *
	resolveFileSymbolName = "resolve_file_symbol_name"

	// Deal with macros that instantiate a different library or scala target name
	//
	// # gazelle:resolve_kind_rewrite_name scala_app %{name} %{name}_lib
	// # gazelle:resolve_kind_rewrite_name classic_scala_app %{name} scala
	// # gazelle:resolve_kind_rewrite_name trumid_scala_test %{name} %{name}_testlib
	// # gazelle:resolve_kind_rewrite_name scala_app_test %{name} %{name}_testlib
	resolveKindRewriteNameDirective = "resolve_kind_rewrite_name"
)

func DirectiveNames() []string {
	return []string{
		resolveConflictsDirective,
		resolveFileSymbolName,
		resolveGlobDirective,
		resolveKindRewriteNameDirective,
		resolveWithDirective,
		scalaDebugDirective,
		scalaLogLevelDirective,
		scalaDepsCleanerDirective,
		scalaFixWildcardImportDirective,
		scalaGenerateBuildFilesDirective,
		scalaRuleDirective,
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
	generateBuildFiles     bool
	logger                 zerolog.Logger
	logLevel               zerolog.Level
}

// New initializes a new Config.
func New(logger zerolog.Logger, universe resolver.Universe, config *config.Config, rel string) *Config {
	return &Config{
		logLevel:          zerolog.DebugLevel,
		logger:            logger,
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
func GetOrCreate(logger zerolog.Logger, universe resolver.Universe, config *config.Config, rel string) *Config {
	var cfg *Config
	if existingExt, ok := config.Exts[scalaLangName]; ok {
		cfg = existingExt.(*Config).clone(config, rel)
	} else {
		cfg = New(logger.With().Str("rel", rel).Logger(), universe, config, rel)
	}
	config.Exts[scalaLangName] = cfg
	return cfg
}

// clone copies this config to a new one.
func (c *Config) clone(config *config.Config, rel string) *Config {
	clone := New(c.logger, c.universe, config, rel)

	clone.logLevel = c.logLevel
	clone.generateBuildFiles = c.generateBuildFiles

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

// Logger returns the child logger set to the configured level
func (c *Config) Logger(logger zerolog.Logger) zerolog.Logger {
	return logger.Level(c.logLevel)
}

// Config returns the parent gazelle configuration
func (c *Config) Config() *config.Config {
	return c.config
}

// Rel returns the parent gazelle relative path
func (c *Config) Rel() string {
	return c.rel
}

func (c *Config) shouldKeep(expr build.Expr, dep *resolver.ImportLabel) bool {
	for _, provider := range c.universe.SymbolProviders() {
		if provider.CanProvide(dep, expr, c.universe.GetKnownRule) {
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
		case scalaLogLevelDirective:
			if err := c.parseScalaLogLevelDirective(d); err != nil {
				return err
			}
		case scalaDebugDirective:
			if err := c.parseScalaAnnotation(d); err != nil {
				return err
			}
		case scalaGenerateBuildFilesDirective:
			if err := c.parseScalaGenerateBuildFilesDirective(d); err != nil {
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
		spec := &fixWildcardImportSpec{
			filenamePattern: filenamePattern,
			importPattern:   *intent,
		}
		c.fixWildcardImportSpecs = append(c.fixWildcardImportSpecs, spec)
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

func (c *Config) parseScalaGenerateBuildFilesDirective(d rule.Directive) error {
	val, err := strconv.ParseBool(d.Value)
	if err != nil {
		return fmt.Errorf("parsing %s: %v", scalaGenerateBuildFilesDirective, err)
	}
	c.generateBuildFiles = val
	return nil
}

func (c *Config) parseScalaLogLevelDirective(d rule.Directive) error {
	level, err := zerolog.ParseLevel(d.Value)
	if err != nil {
		return fmt.Errorf("invalid %v: %v", scalaLogLevelDirective, err)
	}
	c.logLevel = level
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
		r = scalarule.NewConfig(c.Logger(c.logger), c.config, name)
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

func (c *Config) shouldAnnotateDeps() bool {
	_, ok := c.annotations[DebugDeps]
	return ok
}

func (c *Config) shouldAnnotateDepLabelOrigin() bool {
	_, ok := c.annotations[DebugDepLabelOrigin]
	return ok
}

func (c *Config) ShouldAnnotateRule() bool {
	_, ok := c.annotations[DebugRule]
	return ok
}

func (c *Config) GenerateBuildFiles() bool {
	return c.generateBuildFiles
}

func (c *Config) depSuffixComment(imp *resolver.Import) *build.Comment {
	if c.shouldAnnotateDepLabelOrigin() {
		return &build.Comment{Token: fmt.Sprintf("# %v (%s %s)", imp.Kind, imp.Symbol.Provider, imp.Symbol.Name)}
	}
	return &build.Comment{Token: fmt.Sprintf("# %v", imp.Kind)}
}

// ShouldFixWildcardImport tests whether the given symbol name pattern
// should be resolved within the scope of the given filename pattern.
// resolveFileSymbolNameSpecs represent a whitelist; if no patterns match, false
// is returned.
func (c *Config) ShouldFixWildcardImport(filename, wimp string) bool {
	// log.Printf("Checking wildcard import spec: %s %s", filename, wimp)

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

func (c *Config) Imports(imports resolver.ImportMap, r *rule.Rule, attrName string, from label.Label) {
	c.ruleAttrMergeDeps(imports, r, attrName, from)
}

func (c *Config) Exports(exports resolver.ImportMap, r *rule.Rule, attrName string, from label.Label) {
	c.ruleAttrMergeDeps(exports, r, attrName, from)
}

func (c *Config) ruleAttrMergeDeps(
	imports resolver.ImportMap,
	r *rule.Rule,
	attrName string,
	from label.Label,
) {
	// gather the list of dependency labels from the import list
	labels := imports.Deps(c.MaybeRewrite(r.Kind(), from))

	// initialize a map with all true values.  Apply depsCleaners that can set
	// those to false if they are not wanted deps.
	deps := make(map[label.Label]bool)
	for _, l := range labels {
		deps[l.Label] = true
	}
	for _, impl := range c.depsCleaners {
		impl.CleanDeps(deps, r, from)
	}

	// Merge the current list against the new incoming ones.  If no deps remain,
	// delete the attr.
	next := c.mergeDeps(r.Attr(attrName), deps, labels, attrName, from)
	if len(next.List) > 0 {
		r.SetAttr(attrName, next)
	} else {
		r.DelAttr(attrName)
	}
}

// mergeDeps filters out a `deps` list.  Extries are removed from the
// list if they can be parsed as dependency labels that have a provider.  Others
// types of expressions are left as-is.  Dependency labels that have no known
// provider are also left as-is.
func (c *Config) mergeDeps(attrValue build.Expr, deps map[label.Label]bool, importLabels map[label.Label]*resolver.ImportLabel, attrName string, from label.Label) *build.ListExpr {
	var src *build.ListExpr
	if attrValue != nil {
		if current, ok := attrValue.(*build.ListExpr); ok {
			// the value of 'deps' is currently a list, use it.
			src = current
		}
	}
	if src == nil {
		src = new(build.ListExpr)
	}

	var dst = new(build.ListExpr)
	for _, expr := range src.List {
		// try and parse the expression as a label
		dep := labelFromDepExpr(expr)

		// if it wasn't a label, just leave it be (copy it to dst).
		if dep == label.NoLabel {
			dst.List = append(dst.List, expr)
			continue
		}

		// does it have a '# keep' directive?
		if rule.ShouldKeep(expr) {
			// if it does have a keep directive, was the 'keep' unnecessary?  If
			// we have this dep in the incoming deps list, that annotation isn't
			// needed.  Skip it (we'll make a new build.StringExpr in the next loop)
			if want, exists := deps[dep]; exists {
				if want {
					log.Printf(`%v: in attr %q, "%v" does not need a '# keep' directive (fixed)`, attrName, from, dep)
				}
				continue
			}
			// the expression is still wanted
			dst.List = append(dst.List, expr)
			continue
		}

		imp, ok := importLabels[dep]
		if !ok {
			dst.List = append(dst.List, expr)
			continue
		}

		// do we have a known provider for the dependency?  If not, this
		// dependency is not "managed", so leave it alone.
		if !c.shouldKeep(expr, imp) {
			dst.List = append(dst.List, expr)
			continue
		}

		// all other remaining deps should be added in the next loop
	}

	// add managed deps that are wanted
	for dep, want := range deps {
		if !want {
			continue
		}
		imp := importLabels[dep]
		depExpr := &build.StringExpr{Value: dep.String()}
		if c.shouldAnnotateDeps() {
			depExpr.Suffix = append(depExpr.Suffix, depCommentFor(imp.Import))
		}
		comment := c.depSuffixComment(imp.Import)
		if comment != nil {
			depExpr.Suffix = append(depExpr.Suffix, *comment)
		}

		dst.List = append(dst.List, depExpr)
	}

	return dst
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
	case "deps":
		return DebugDeps
	case "rule":
		return DebugRule
	case "dep_label_origin":
		return DebugDepLabelOrigin
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

func AnnotateImports(imports resolver.ImportMap, comments *build.Comments, prefix string) {
	for _, key := range imports.Keys() {
		imp, _ := imports.Get(key)
		comment := setCommentPrefix(imp.Comment(), prefix)
		// log.Printf("%d: %v", i, comment.Token)
		comments.Before = append(comments.Before, comment)
	}
}

func depCommentFor(imp *resolver.Import) build.Comment {
	source := ""
	if imp.Source != nil {
		source = imp.Source.Filename
	}
	token := fmt.Sprintf("# imp=%s, type=%v, provider=%s, label=%v, kind=%v, source=%s", imp.Imp, imp.Symbol.Type, imp.Symbol.Provider, imp.Symbol.Label, imp.Kind, source)
	return build.Comment{
		Token: token,
	}
}

func setCommentPrefix(comment build.Comment, prefix string) build.Comment {
	comment.Token = "# " + prefix + strings.TrimSpace(strings.TrimPrefix(comment.Token, "#"))
	return comment
}
