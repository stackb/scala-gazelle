package scala

import (
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
)

type scalaRuleContext struct {
	// the parent config
	scalaConfig *scalaConfig
	// from is the label for the rule.
	from label.Label
	// rule (lowercase) is the parent gazelle rule
	rule *rule.Rule
	// scope is a map of symbols that are outside the rule.
	scope resolver.Scope
	// the import resolver to which we chain to when self-imports are not
	// matched.
	resolver resolver.SymbolResolver
}

type scalaRule struct {
	// Rule is the pb representation
	pb *sppb.Rule
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
		ctx:     ctx,
		exports: make(map[string]resolve.ImportSpec),
	}

	if !isBinaryRule(ctx.rule.Kind()) {
		for _, file := range scalaRule.pb.Files {
			scalaRule.putExports(file)
		}
	}

	return scalaRule
}

// ResolveSymbol implements the resolver.SymbolResolver interface.
func (r *scalaRule) ResolveSymbol(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.Symbol, error) {
	if symbol, ok := r.ctx.scope.GetSymbol(imp); ok {
		return symbol, nil
	}
	return r.ctx.resolver.ResolveSymbol(c, ix, from, lang, imp)
}

// Imports implements part of the scalarule.Rule interface.
func (r *scalaRule) Imports() resolver.ImportMap {
	imports := resolver.NewImportMap()
	impLang := scalaLangName

	// direct
	for _, file := range r.pb.Files {
		r.fileImports(file, imports)
	}

	// if this rule has a main_class
	if mainClass := r.ctx.rule.AttrString("main_class"); mainClass != "" {
		imports.Put(resolver.NewMainClassImport(mainClass))
	}

	// Initialize a list of symbols to find implicits for from all known
	// imports. Include all symbols that are defined in the rule too (a
	// gazelle:resolve_with directive should apply to them too).
	required := collections.StringStack(imports.Keys())
	for _, export := range r.Exports() {
		required = append(required, export.Imp)
	}

	// Gather implicit imports transitively.
	for !required.IsEmpty() {
		src, _ := required.Pop()
		for _, dst := range r.ctx.scalaConfig.getImplicitImports(impLang, src) {
			required.Push(dst)
			imports.Put(resolver.NewImplicitImport(dst, src))
		}
	}

	return imports
}

// fileImports gathers needed imports for the given file.
func (r *scalaRule) fileImports(file *sppb.File, imports resolver.ImportMap) {
	var scopes []resolver.Scope

	// gather import scopes
	for _, imp := range file.Imports {
		if wimp, ok := isWildcardImport(imp); ok {
			if scope, ok := r.ctx.scope.GetScope(wimp); ok {
				scopes = append(scopes, scope)
			} else {
				log.Printf("%s | warning: wildcard import scope not found: %s", r.ctx.from, wimp)
			}
		} else {
			imports.Put(resolver.NewDirectImport(imp, file))
		}
	}
	// gather package scopes
	for _, pkg := range file.Packages {
		if scope, ok := r.ctx.scope.GetScope(pkg); ok {
			scopes = append(scopes, scope)
		} else {
			log.Printf("%s | warning: package scope not found: %s", r.ctx.from, pkg)
		}
	}

	// add in outer scope
	scopes = append(scopes, r.ctx.scope)
	// build final scope used to resolve names in the file.
	scope := resolver.NewChainScope(scopes...)

	// resolve extends clauses in the file.  While these are probably duplicated
	// in the 'Names' slice, do it anyway.
	for token, extends := range file.Extends {
		parts := strings.SplitN(token, " ", 2)
		if len(parts) != 2 {
			log.Fatalf("invalid extends token: %q: should have form '(class|interface|object) com.foo.Bar' ", token)
		}

		// kind := parts[0] // kind not used
		name := parts[1]

		for _, imp := range extends.Classes {
			if sym, ok := scope.GetSymbol(imp); ok {
				imports.Put(resolver.NewExtendsImport(sym.Name, file, name, sym))
			} else {
				log.Printf("%s | %s: extends name not found: %s", r.ctx.from, file.Filename, name)
			}
		}
	}

	// resolve symbols named in the file.  For each one we find, add an import.
	for _, name := range file.Names {
		if !r.ctx.scalaConfig.shouldResolveName(r.ctx.from, file, name) {
			continue
		}
		if sym, ok := scope.GetSymbol(name); ok {
			imports.Put(resolver.NewResolvedSymbolImport(sym.Name, file, name, sym))
		} else {
			log.Printf("%s | %s: name not found: %s", r.ctx.from, file.Filename, name)
		}
	}

}

func (r *scalaRule) Files() []*sppb.File {
	return r.pb.Files
}

// Exports implements part of the scalarule.Rule interface.
func (r *scalaRule) Exports() []resolve.ImportSpec {
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

func isWildcardImport(imp string) (string, bool) {
	if !strings.HasSuffix(imp, "._") {
		return "", false
	}
	return imp[:len(imp)-len("._")], true
}

func isBinaryRule(kind string) bool {
	return strings.Contains(kind, "binary") || strings.Contains(kind, "test")
}
