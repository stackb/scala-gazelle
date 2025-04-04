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
	"github.com/rs/zerolog"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/wildcardimport"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

const (
	debugNameNotFound        = false
	debugUnresolved          = false
	debugExtendsNameNotFound = false
	debugFileScope           = false
)

type scalaRuleContext struct {
	// the parent config
	scalaConfig *scalaconfig.Config
	// rule (lowercase) is the parent gazelle rule
	rule *rule.Rule
	// scope is a map of symbols that are outside the rule.
	scope resolver.Scope
	// the global import resolver
	resolver resolver.SymbolResolver
}

type scalaRule struct {
	// logger instance
	logger zerolog.Logger
	// Rule is the pb representation
	pb *sppb.Rule
	// files is a list of files, copied from pb.Files but sorted again
	files []*sppb.File
	// ctx is the rule context
	ctx *scalaRuleContext
	// exports keyed by their import
	exports map[string]resolve.ImportSpec
}

var bazel = "bazel"

func init() {
	if bazelExe, ok := os.LookupEnv("SCALA_GAZELLE_BAZEL_EXECUTABLE"); ok {
		bazel = bazelExe
	}
}

func newScalaRule(
	logger zerolog.Logger,
	ctx *scalaRuleContext,
	rule *sppb.Rule,
) *scalaRule {
	scalaRule := &scalaRule{
		logger:  logger,
		pb:      rule,
		files:   rule.Files,
		ctx:     ctx,
		exports: make(map[string]resolve.ImportSpec),
	}

	sort.Slice(scalaRule.files, func(i, j int) bool {
		a := scalaRule.files[i]
		b := scalaRule.files[j]
		return a.Filename < b.Filename
	})

	if !isBinaryRule(ctx.rule.Kind()) {
		for _, file := range scalaRule.files {
			scalaRule.putExports(file)
		}
	}

	return scalaRule
}

// ResolveExports performs symbol resolution for exports of the rule.
func (r *scalaRule) ResolveExports(rctx *scalarule.ResolveContext) resolver.ImportMap {
	exports := r.Exports(rctx.From)

	//
	// part 1: resolve any unsettled imports and populate the transitive stack.
	//
	for _, imp := range exports.Values() {
		if imp.Error != nil {
			continue
		}
		if symbol, ok := r.ResolveSymbol(rctx.Config, rctx.RuleIndex, rctx.From, scalaLangName, imp.Imp); ok {
			imp.Symbol = symbol
		} else {
			r.logger.Print(r.pb.Label + ": unresolved export: " + imp.Imp)
			imp.Error = resolver.ErrSymbolNotFound
		}
	}

	return exports
}

// ResolveImports performs symbol resolution for imports of the rule.
func (r *scalaRule) ResolveImports(rctx *scalarule.ResolveContext) resolver.ImportMap {
	imports := r.Imports(rctx.From)
	sc := scalaconfig.Get(rctx.Config)

	for _, imp := range imports.Values() {
		if imp.Error != nil {
			continue
		}
		if symbol, ok := r.ResolveSymbol(rctx.Config, rctx.RuleIndex, rctx.From, scalaLangName, imp.Imp); ok {
			imp.Symbol = symbol
			if len(imp.Symbol.Conflicts) > 0 {
				if resolved, ok := sc.ResolveConflict(rctx.Rule, imports, imp, imp.Symbol); ok {
					imp.Symbol = resolved
					r.logger.Debug().
						Msgf("conflict resolved import %s to %s", imp.Imp, symbol.String())
				} else {
					message := resolver.SymbolConfictMessage(imp.Symbol, imp, rctx.From)
					r.logger.Warn().Msg(message)
					fmt.Println(message)
					r.logger.Debug().
						Msgf("resolved still-conflicted import %s to %s", imp.Imp, symbol.String())
				}
			} else {
				r.logger.Debug().
					Msgf("resolved unconflicted import %s to %s", imp.Imp, symbol.String())
			}
		} else {
			r.logger.Print(r.pb.Label + ": unresolved import: " + imp.Imp)
			imp.Error = resolver.ErrSymbolNotFound
		}
	}

	return imports
}

// ResolveSymbol implements the resolver.SymbolResolver interface.
func (r *scalaRule) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.Symbol, bool) {
	return r.ctx.resolver.ResolveSymbol(c, ix, from, lang, imp)
}

// Imports implements part of the scalarule.Rule interface.
func (r *scalaRule) Imports(from label.Label) resolver.ImportMap {
	imports := resolver.NewImportMap()
	impLang := scalaLangName

	// if this rule has a main_class
	if mainClass := r.ctx.rule.AttrString("main_class"); mainClass != "" {
		imports.Put(resolver.NewMainClassImport(mainClass))
	}

	// direct
	for _, file := range r.files {
		r.fileImports(imports, file, from)
	}

	// semantic add in semantic imports after direct ones to minimize the delta
	// between running gazelle with and without semanticdb info.
	for _, file := range r.files {
		r.fileSemanticImports(imports, file, from)
	}

	// Initialize a list of symbols to find implicits for from all known
	// imports. Include all symbols that are defined in the rule too (a
	// gazelle:resolve_with directive should apply to them too).
	required := collections.StringStack(imports.Keys())
	for _, export := range r.Provides() {
		required = append(required, export.Imp)
	}

	// Gather implicit imports transitively.
	for !required.IsEmpty() {
		src, _ := required.Pop()
		for _, dst := range r.ctx.scalaConfig.GetImplicitImports(impLang, src) {
			required.Push(dst)
			imports.Put(resolver.NewImplicitImport(dst, src))
		}
	}

	return imports
}

// Exports implements part of the scalarule.Rule interface.
func (r *scalaRule) Exports(from label.Label) resolver.ImportMap {
	exports := resolver.NewImportMap()

	for _, file := range r.files {
		r.fileExports(file, exports, from)
	}

	return exports
}

// fileExports gathers exports for the given file.
func (r *scalaRule) fileExports(file *sppb.File, exports resolver.ImportMap, from label.Label) {

	putExport := resolver.PutImportIfNotSelf(exports, from)

	var scopes []resolver.Scope
	direct := resolver.NewTrieScope()

	// add in outer scope
	scopes = append(scopes, r.ctx.scope, direct)
	// build final scope used to resolve names in the file.
	scope := resolver.NewChainScope(scopes...)

	if debugFileScope {
		r.logger.Print(r.infof("%s scope:\n%s", file.Filename, scope.String()))
	}

	// resolve extends clauses in the file.  While these are probably duplicated
	// in the 'Names' slice, do it anyway.
	tokens := extendsKeysSorted(file.Extends)
	for _, token := range tokens {
		extends := file.Extends[token]
		parts := strings.SplitN(token, " ", 2)
		if len(parts) != 2 {
			log.Fatalf("invalid extends token: %q: should have form '(class|interface|object) com.foo.Bar' ", token)
		}

		name := parts[1] // note: parts[0] is the 'kind'

		// assume the name is fully-qualified so resolve it from the "root"
		// scope rather than involving package scopes.
		resolved, resolvedOK := r.ctx.scope.GetSymbol(name)

		for _, imp := range extends.Classes {
			if sym, ok := scope.GetSymbol(imp); ok {
				// if the symbol has conflicts, don't export it
				if len(sym.Conflicts) > 0 {
					r.logger.Print(r.warnf("%s: %q extends %q, but symbol %q is conflicted", file.Filename, name, imp, imp))
				} else {
					putExport(resolver.NewExtendsImport(sym.Name, file, name, sym))
					if resolvedOK && resolved != sym {
						resolved.Require(sym)
					}
				}
			} else {
				putExport(resolver.NewExtendsImport(imp, file, name, nil))
				r.logger.Print(r.warnf("%s: %q extends %q, but symbol %q is unknown", file.Filename, name, imp, imp))
			}
		}
	}

}

// fileSemanticImports gathers needed semantic imports for the given file.
func (r *scalaRule) fileSemanticImports(imports resolver.ImportMap, file *sppb.File, from label.Label) {

	putImport := resolver.PutImportIfNotSelf(imports, from)

	for _, name := range file.SemanticImports {
		imp := resolver.NewSemanticImport(name, file)
		if sym, ok := r.ctx.scope.GetSymbol(imp.Imp); ok {
			imp.Symbol = sym
		}
		putImport(imp)
	}

}

// fileImports gathers needed imports for the given file.
func (r *scalaRule) fileImports(imports resolver.ImportMap, file *sppb.File, from label.Label) {

	var scopes []resolver.Scope
	direct := resolver.NewTrieScope()

	putImport := resolver.PutImportIfNotSelf(imports, from)

	// gather direct imports and import scopes
	for _, name := range file.Imports {
		if wimp, ok := resolver.IsWildcardImport(name); ok {
			filename := filepath.Join(r.ctx.scalaConfig.Rel(), file.Filename)
			if r.ctx.scalaConfig.ShouldFixWildcardImport(filename, name) {
				symbolNames, err := r.fixWildcardImport(filename, wimp)
				if err != nil {
					log.Fatalf("fixing wildcard imports for %s (%s): %v", file.Filename, wimp, err)
				}
				for _, symName := range symbolNames {
					fqn := wimp + "." + symName
					if sym, ok := r.ctx.scope.GetSymbol(fqn); ok {
						putImport(resolver.NewResolvedNameImport(sym.Name, file, fqn, sym))
					} else {
						r.logger.Printf("%s: warning: unresolved fix wildcard import: symbol %q: was not found' (%s)", r.pb.Label, name, file.Filename)
					}
				}
			}

			// collect the (package) symbol for import
			if sym, ok := r.ctx.scope.GetSymbol(name); ok {
				// log.Printf("%v: WARN: resolved name import: %v %v %v", from, sym.Name, file.Filename, sym.Label)
				putImport(resolver.NewResolvedNameImport(sym.Name, file, name, sym))
			} else {
				r.logger.Printf("%s: warning: unresolved wildcard import: symbol %q: was not found' (%s)", r.pb.Label, name, file.Filename)
				imp := resolver.NewDirectImport(name, file)
				putImport(imp)
			}

			// collect the scope
			if scope, ok := r.ctx.scope.GetScope(wimp); ok {
				scopes = append(scopes, scope)
			} else if debugNameNotFound {
				log.Printf("%s | warning: wildcard import scope not found: %s", r.pb.Label, wimp)
			}
		} else {
			imp := resolver.NewDirectImport(name, file)
			if sym, ok := r.ctx.scope.GetSymbol(name); ok {
				imp.Symbol = sym
				direct.Put(importBasename(name), sym)
			} else if debugNameNotFound {
				log.Printf("%s | warning: direct symbol not found: %s", r.pb.Label, name)
			}
			putImport(imp)
		}
	}

	// gather package scopes
	for _, pkg := range file.Packages {
		if scope, ok := r.ctx.scope.GetScope(pkg); ok {
			scopes = append(scopes, scope)
		} else if debugNameNotFound {
			log.Printf("%s | warning: package scope not found: %s", r.pb.Label, pkg)
		}
	}

	// add in outer scope
	scopes = append(scopes, r.ctx.scope, direct)
	// build final scope used to resolve names in the file.
	scope := resolver.NewChainScope(scopes...)

	if debugFileScope {
		r.logger.Print(r.infof("%s scope:\n%s", file.Filename, scope.String()))
	}

	// resolve extends clauses in the file.  While these are probably duplicated
	// in the 'Names' slice, do it anyway.
	tokens := extendsKeysSorted(file.Extends)
	for _, token := range tokens {
		extends := file.Extends[token]
		parts := strings.SplitN(token, " ", 2)
		if len(parts) != 2 {
			log.Panicf("invalid extends token: %q: should have form '(class|interface|object) com.foo.Bar' ", token)
		}

		// kind := parts[0]
		name := parts[1]

		// assume the name if fully-qualified, so resolve it from the "root"
		// scope rather than involving package scopes.
		resolved, resolvedOK := r.ctx.scope.GetSymbol(name)
		if !resolvedOK {
			r.logger.Print(r.warnf("%s: extends symbol not found: %s", file.Filename, name))
		}

		for _, imp := range extends.Classes {
			if sym, ok := scope.GetSymbol(imp); ok {
				putImport(resolver.NewExtendsImport(sym.Name, file, name, sym))
				if resolvedOK && resolved != sym {
					resolved.Require(sym)
				}
			} else {
				r.logger.Print(r.warnf("%s: extends symbol not found: %s", file.Filename, imp))

				if debugExtendsNameNotFound {
					putImport(resolver.NewExtendsImport(imp, file, name, nil))
				}
			}
		}
	}

}

// Files implements part of the scalarule.Rule interface.
func (r *scalaRule) Files() []*sppb.File {
	return r.files
}

// Provides implements part of the scalarule.Rule interface.
func (r *scalaRule) Provides() []resolve.ImportSpec {
	exports := make([]resolve.ImportSpec, 0, len(r.exports))
	for _, v := range r.exports {
		exports = append(exports, v)
	}

	sort.Slice(exports, func(i, j int) bool {
		a := exports[i]
		b := exports[j]
		return a.Imp < b.Imp
	})

	return exports
}

func (r *scalaRule) putExports(file *sppb.File) {
	for _, imp := range file.Classes {
		r.putExport(imp)
	}
	for _, imp := range file.Objects {
		r.putExport(imp)
	}
	for _, imp := range file.Traits {
		r.putExport(imp)
	}
	for _, imp := range file.Types {
		r.putExport(imp)
	}
	for _, imp := range file.Vals {
		r.putExport(imp)
	}
}

func (r *scalaRule) putExport(imp string) {
	r.exports[imp] = resolve.ImportSpec{Imp: imp, Lang: scalaLangName}
}

func (r *scalaRule) fixWildcardImport(filename, wimp string) ([]string, error) {
	fixer := wildcardimport.NewFixer(&wildcardimport.FixerOptions{
		BazelExecutable: bazel,
	})

	absFilename := filepath.Join(wildcardimport.GetBuildWorkspaceDirectory(), filename)
	ruleLabel := label.New("", r.ctx.scalaConfig.Rel(), r.ctx.rule.Name()).String()
	symbols, err := fixer.Fix(ruleLabel, absFilename, wimp)
	if err != nil {
		return nil, err
	}

	return symbols, nil
}

func (r *scalaRule) debugf(format string, args ...any) string {
	return r.printf("DEBUG", format, args...)
}

func (r *scalaRule) infof(format string, args ...any) string {
	return r.printf("INFO", format, args...)
}

func (r *scalaRule) warnf(format string, args ...any) string {
	return r.printf("WARN", format, args...)
}

func (r *scalaRule) printf(level, format string, args ...any) string {
	return fmt.Sprintf(level+" ["+r.ctx.scalaConfig.Rel()+": "+format, args...)
}

func ImportsPutIfNotSelfImport(imports resolver.ImportMap, repo, rel, ruleName string) func(*resolver.Import) {
	return func(imp *resolver.Import) {
		if !resolver.IsSelfImport(imp.Symbol, repo, rel, ruleName) {
			imports.Put(imp)
		}
	}
}

func isBinaryRule(kind string) bool {
	return strings.Contains(kind, "binary") || strings.Contains(kind, "test")
}

func importBasename(imp string) string {
	index := strings.LastIndex(imp, ".")
	if index == -1 {
		return imp
	}
	return imp[index+1:]
}

func extendsKeysSorted(collection map[string]*sppb.ClassList) []string {
	keys := make([]string, 0, len(collection))
	for token := range collection {
		keys = append(keys, token)
	}
	sort.Strings(keys)
	return keys
}
