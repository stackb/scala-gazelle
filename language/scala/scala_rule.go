package scala

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

type ScalaRule struct {
	*rule.Rule
	From  label.Label
	Files []*sppb.File

	next     resolver.KnownImportResolver
	registry resolver.KnownImportRegistry
	// requiredTypes is a mapping from the required type to the symbol that needs
	// it. for example, if 'class Foo requiredTypes Bar', "Bar" is the map key and
	// "Foo" will be the value.
	requiredTypes map[string][]string
}

func NewScalaRule(registry resolver.KnownImportRegistry, next resolver.KnownImportResolver, r *rule.Rule, from label.Label, files []*sppb.File) *ScalaRule {
	scalaRule := &ScalaRule{
		Rule:          r,
		From:          from,
		Files:         files,
		next:          next,
		registry:      registry,
		requiredTypes: make(map[string][]string),
	}
	scalaRule.addFiles(files...)
	return scalaRule
}

func (r *ScalaRule) addFiles(files ...*sppb.File) {
	for _, file := range files {
		r.addFromFile(file)
	}
}

func (r *ScalaRule) addFromFile(file *sppb.File) {
	for _, imp := range file.Classes {
		r.putKnownImport(imp, sppb.ImportType_CLASS)
	}
	for _, imp := range file.Objects {
		r.putKnownImport(imp, sppb.ImportType_OBJECT)
	}
	for _, imp := range file.Traits {
		r.putKnownImport(imp, sppb.ImportType_TRAIT)
	}
	for _, imp := range file.Types {
		r.putKnownImport(imp, sppb.ImportType_TYPE)
	}
	for _, imp := range file.Vals {
		r.putKnownImport(imp, sppb.ImportType_VALUE)
	}
	// for _, imp := range file.Packages {
	// 	r.putKnownImport(imp, sppb.ImportType_PACKAGE)
	// }
	for token, extends := range file.Extends {
		r.putExtends(token, extends)
	}
	for _, imp := range file.Imports {
		r.putFileImport(imp)
	}
}

func (r *ScalaRule) putFileImport(imp string) {
	// r.imports.Put(imp)
}

func (r *ScalaRule) putKnownImport(imp string, impType sppb.ImportType) {
	// since we don't need to resolve same-rule symbols to a different label,
	// record all imports as label.NoLabel!
	r.registry.PutKnownImport(resolver.NewKnownImport(impType, imp, "self-import", label.NoLabel))
}

func (r *ScalaRule) putExtends(token string, types *sppb.ClassList) {
	parts := strings.SplitN(token, " ", 2)
	if len(parts) != 2 {
		log.Fatalf("invalid extends token: %q: should have form '(class|interface|object) com.foo.Bar' ", token)
	}

	kind := parts[0]
	symbol := parts[1]

	r.putKindExtends(kind, symbol, types)
}

func (r *ScalaRule) putKindExtends(kind, symbol string, types *sppb.ClassList) {
	switch kind {
	case "class":
		r.putClassExtends(symbol, types)
	case "interface":
		r.putInterfaceExtends(symbol, types)
	case "object":
		r.putObjectExtends(symbol, types)
	}
}

func (r *ScalaRule) putClassExtends(imp string, types *sppb.ClassList) {
	r.putRequiredTypes(imp, types)
}

func (r *ScalaRule) putInterfaceExtends(imp string, types *sppb.ClassList) {
	r.putRequiredTypes(imp, types)
}

func (r *ScalaRule) putObjectExtends(imp string, types *sppb.ClassList) {
	r.putRequiredTypes(imp, types)
}

func (r *ScalaRule) putRequiredTypes(imp string, types *sppb.ClassList) {
	for _, dst := range types.Classes {
		r.putRequiredType(imp, dst)
	}

}

// ResolveKnownImport implements the resolver.KnownImportResolver interface
func (r *ScalaRule) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*resolver.KnownImport, error) {
	if known, ok := r.registry.GetKnownImport(imp); ok {
		return known, nil
	}
	return r.next.ResolveKnownImport(c, ix, from, lang, imp)
}

func (r *ScalaRule) putRequiredType(src, dst string) {
	r.requiredTypes[dst] = append(r.requiredTypes[dst], src)
}

func (r *ScalaRule) Imports(sc *scalaConfig) resolver.ImportMap {
	imports := resolver.NewImportMap()
	impLang := scalaLangName

	// direct
	for _, file := range r.Files {
		for _, imp := range file.Imports {
			imports.Put(resolver.NewDirectImport(imp, file))
		}
	}

	// if this rule has a main_class
	if mainClass := r.AttrString("main_class"); mainClass != "" {
		imports.Put(resolver.NewMainClassImport(mainClass))
	}

	// add import required from extends clauses
	for imp, src := range r.requiredTypes {
		imports.Put(resolver.NewExtendsImport(imp, src[0])) // use first occurrence as source arg
	}

	// gather implicit imports
	transitive := make(collections.StringStack, 0)
	for src := range imports {
		for _, dst := range sc.getImplicitImports(impLang, src) {
			transitive.Push(dst)
			imports.Put(resolver.NewImplicitImport(dst, src))
		}
	}
	for !transitive.IsEmpty() {
		src, _ := transitive.Pop()
		for _, dst := range sc.getImplicitImports(impLang, src) {
			transitive.Push(dst)
			imports.Put(resolver.NewImplicitImport(dst, src))
		}
	}

	return imports
}
