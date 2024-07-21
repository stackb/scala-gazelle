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

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const (
	debugSelfImports         = false
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
	// Rule is the pb representation
	pb *sppb.Rule
	// files is a list of files, copied from pb.Files but sorted again
	files []*sppb.File
	// ctx is the rule context
	ctx *scalaRuleContext
	// exports keyed by their import
	exports map[string]resolve.ImportSpec
}

func newScalaRule(
	ctx *scalaRuleContext,
	rule *sppb.Rule,
) *scalaRule {
	scalaRule := &scalaRule{
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
	exports := r.Exports()

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
			if debugUnresolved {
				log.Println("unresolved export:", imp)
			}
			imp.Error = resolver.ErrSymbolNotFound
		}
	}

	return exports
}

// ResolveImports performs symbol resolution for imports of the rule.
func (r *scalaRule) ResolveImports(rctx *scalarule.ResolveContext) resolver.ImportMap {
	imports := r.Imports()
	sc := scalaconfig.Get(rctx.Config)

	transitive := newImportSymbols()

	//
	// part 1: resolve any unsettled imports and populate the transitive stack.
	//
	for _, imp := range imports.Values() {
		if imp.Error != nil {
			continue
		}
		if symbol, ok := r.ResolveSymbol(rctx.Config, rctx.RuleIndex, rctx.From, scalaLangName, imp.Imp); ok {
			imp.Symbol = symbol
		} else {
			if debugUnresolved {
				log.Println("unresolved import:", imp)
			}
			imp.Error = resolver.ErrSymbolNotFound
		}
		if imp.Symbol != nil {
			transitive.Push(imp, imp.Symbol)
		}
	}

	//
	// part 2: process each symbol and address conflicts, transitively.
	//
	for !transitive.IsEmpty() {
		item, _ := transitive.Pop()

		if len(item.sym.Conflicts) > 0 {
			if resolved, ok := sc.ResolveConflict(rctx.Rule, imports, item.imp, item.sym, rctx.From); ok {
				if resolved != nil {
					item.imp.Symbol = resolved
				} else {
					log.Println("deleting import!", item.imp.Imp)
					delete(imports, item.imp.Imp)
					continue // skip this item if conflict strategy says "ok" but returns nil (it's a wildcard import)
				}
			} else {
				fmt.Println(resolver.SymbolConfictMessage(item.sym, item.imp, rctx.From))
			}
		}

		// do something here to augment requires?

		for _, req := range item.sym.Requires {
			if _, ok := imports[req.Name]; ok {
				continue
			}
			imports.Put(resolver.NewTransitiveImport(req.Name, item.sym.Name, req))
			transitive.Push(item.imp, req)
		}
	}

	r.pb.ResolvedImports = imports.ProtoList()

	return imports
}

// ResolveSymbol implements the resolver.SymbolResolver interface.
func (r *scalaRule) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.Symbol, bool) {
	return r.ctx.resolver.ResolveSymbol(c, ix, from, lang, imp)
}

// Imports implements part of the scalarule.Rule interface.
func (r *scalaRule) Imports() resolver.ImportMap {
	imports := resolver.NewImportMap()
	impLang := scalaLangName

	// if this rule has a main_class
	if mainClass := r.ctx.rule.AttrString("main_class"); mainClass != "" {
		imports.Put(resolver.NewMainClassImport(mainClass))
	}

	// direct
	for _, file := range r.files {
		r.fileImports(imports, file)
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
func (r *scalaRule) Exports() resolver.ImportMap {
	exports := resolver.NewImportMap()

	for _, file := range r.files {
		r.fileExports(file, exports)
	}

	r.pb.ResolvedImports = exports.ProtoList()

	return exports
}

// fileExports gathers needed imports for the given file.
func (r *scalaRule) fileExports(file *sppb.File, exports resolver.ImportMap) {
	var scopes []resolver.Scope
	direct := resolver.NewTrieScope()

	putExport := func(imp *resolver.Import) {
		if resolver.IsSelfImport(imp, "", r.ctx.scalaConfig.Rel(), r.ctx.rule.Name()) {
			if debugSelfImports {
				log.Println("skipping export from current", imp.Imp)
			}
			return
		}
		exports.Put(imp)
	}

	// add in outer scope
	scopes = append(scopes, r.ctx.scope, direct)
	// build final scope used to resolve names in the file.
	scope := resolver.NewChainScope(scopes...)

	if debugFileScope {
		log.Printf("%s scope:\n%s", file.Filename, scope.String())
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
				putExport(resolver.NewExtendsImport(sym.Name, file, name, sym))
				if resolvedOK && resolved != sym {
					resolved.Require(sym)
				}
			} else {
				putExport(resolver.NewExtendsImport(imp, file, name, nil))
				if debugExtendsNameNotFound {
					log.Printf("%s | %s: %q extends %q, but symbol %q is unknown", r.pb.Label, file.Filename, name, imp, imp)
				}
			}
		}
	}

}

// fileImports gathers needed imports for the given file.
func (r *scalaRule) fileImports(imports resolver.ImportMap, file *sppb.File) {
	var scopes []resolver.Scope
	direct := resolver.NewTrieScope()

	putImport := func(imp *resolver.Import) {
		if resolver.IsSelfImport(imp, "", r.ctx.scalaConfig.Rel(), r.ctx.rule.Name()) {
			if debugSelfImports {
				log.Println("skipping import from current", imp.Imp)
			}
			return
		}
		imports.Put(imp)
	}

	// gather direct imports and import scopes
	for _, name := range file.Imports {
		if wimp, ok := resolver.IsWildcardImport(name); ok {
			// collect the (package) symbol for import
			if sym, ok := r.ctx.scope.GetSymbol(name); ok {
				putImport(resolver.NewResolvedNameImport(sym.Name, file, name, sym))
			} else {
				if debugUnresolved {
					log.Printf("warning: unresolved wildcard import: symbol %q: was not found' (%s)", name, file.Filename)
				}
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
		log.Printf("%s scope:\n%s", file.Filename, scope.String())
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

		// kind := parts[0]
		name := parts[1]

		// assume the name if fully-qualified, so resolve it from the "root"
		// scope rather than involving package scopes.
		resolved, resolvedOK := r.ctx.scope.GetSymbol(name)
		if !resolvedOK {
			log.Printf("warning: invalid extends token: symbol %q: was not found' ", name)
		}

		for _, imp := range extends.Classes {
			if sym, ok := scope.GetSymbol(imp); ok {
				putImport(resolver.NewExtendsImport(sym.Name, file, name, sym))
				if resolvedOK && resolved != sym {
					resolved.Require(sym)
				}
			} else if debugExtendsNameNotFound {
				putImport(resolver.NewExtendsImport(imp, file, name, nil))
			}
		}
	}

	// resolve symbols named in the file.  For each one we find, add an import.
	for _, name := range file.Names {
		if !r.ctx.scalaConfig.ShouldResolveFileSymbolName(file.Filename, name) {
			continue
		}
		if sym, ok := scope.GetSymbol(name); ok {
			putImport(resolver.NewResolvedNameImport(sym.Name, file, name, sym))
		} else {
			putImport(resolver.NewErrorImport(name, file, "", fmt.Errorf("name not found")))
		}
	}
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

func (r *scalaRule) fixWildcardImports() error {
	return fixWildcardRuleImports(r.ctx.scalaConfig, r.pb)
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

// importSymbol is a pair (import, symbol). If pair.imp.Symbol == pair.sym it
// represents a direct, otherwise pair.sym is a transitive requirement of
// pair.imp.
type importSymbol struct {
	imp *resolver.Import
	sym *resolver.Symbol
}

// importSymbols is a stack of importSymbol pairs.
type importSymbols []*importSymbol

func newImportSymbols() importSymbols {
	return []*importSymbol{}
}

// IsEmpty checks if the stack is empty
func (s *importSymbols) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new pair onto the stack
func (s *importSymbols) Push(imp *resolver.Import, sym *resolver.Symbol) {
	*s = append(*s, &importSymbol{imp, sym})
}

// Pop: remove and return top element of stack, return false if stack is empty
func (s *importSymbols) Pop() (*importSymbol, bool) {
	if s.IsEmpty() {
		return nil, false
	}

	i := len(*s) - 1
	x := (*s)[i]
	*s = (*s)[:i]

	return x, true
}

func extendsKeysSorted(collection map[string]*sppb.ClassList) []string {
	keys := make([]string, 0, len(collection))
	for token := range collection {
		keys = append(keys, token)
	}
	sort.Strings(keys)
	return keys
}
