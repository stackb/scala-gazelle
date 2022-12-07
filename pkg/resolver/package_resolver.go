package resolver

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

type fromFile struct {
	from label.Label
	file *sppb.File
}

// PackageResolver implements KnownImportResolver for a set of files in a
// package. After indexing the symbols in the given files, an additional set of
// imports representing types needed for 'extends' clauses can be obtained.
type PackageResolver struct {
	next     KnownImportResolver
	registry KnownImportRegistry
	// requiredTypes is a mapping from the required type to the symbol that needs
	// it. for example, if 'class Foo requiredTypes Bar', "Bar" is the map key and
	// "Foo" will be the value.
	requiredTypes map[string][]string
}

func NewPackageResolver(next KnownImportResolver) *PackageResolver {
	r := &PackageResolver{
		next:          next,
		registry:      NewKnownImportRegistryTrie(),
		requiredTypes: make(map[string][]string),
	}
	return r
}

func (r *PackageResolver) AddFiles(from label.Label, files ...*sppb.File) {
	for _, file := range files {
		r.addFromFile(&fromFile{from: from, file: file})
	}
}

func (r *PackageResolver) addFromFile(ff *fromFile) {
	for _, imp := range ff.file.Classes {
		r.putKnownImport(ff, imp, sppb.ImportType_CLASS)
	}
	for _, imp := range ff.file.Objects {
		r.putKnownImport(ff, imp, sppb.ImportType_OBJECT)
	}
	for _, imp := range ff.file.Traits {
		r.putKnownImport(ff, imp, sppb.ImportType_TRAIT)
	}
	for _, imp := range ff.file.Vals {
		r.putKnownImport(ff, imp, sppb.ImportType_VALUE)
	}
	for _, imp := range ff.file.Packages {
		r.putKnownImport(ff, imp, sppb.ImportType_PACKAGE)
	}
	for token, extends := range ff.file.Extends {
		r.putExtends(token, extends)
	}
	for _, imp := range ff.file.Imports {
		r.putFileImport(imp)
	}
}

func (r *PackageResolver) putFileImport(imp string) {
	// r.imports.Put(imp)
}

func (r *PackageResolver) putKnownImport(ff *fromFile, imp string, impType sppb.ImportType) {
	r.registry.PutKnownImport(&KnownImport{
		Provider: ff.from.String(),
		Type:     impType,
		Import:   imp,
		Label:    ff.from,
	})
}

func (r *PackageResolver) putExtends(token string, types *sppb.ClassList) {
	parts := strings.SplitN(token, " ", 2)
	if len(parts) != 2 {
		log.Fatalf("invalid extends token: %q: should have form '(class|interface|object) com.foo.Bar' ", token)
	}

	kind := parts[0]
	symbol := parts[1]

	r.putKindExtends(kind, symbol, types)
}

func (r *PackageResolver) putKindExtends(kind, symbol string, types *sppb.ClassList) {
	switch kind {
	case "class":
		r.putClassExtends(symbol, types)
	case "interface":
		r.putInterfaceExtends(symbol, types)
	case "object":
		r.putObjectExtends(symbol, types)
	}
}

func (r *PackageResolver) putClassExtends(imp string, types *sppb.ClassList) {
	r.putRequiredTypes(imp, types)
}

func (r *PackageResolver) putInterfaceExtends(imp string, types *sppb.ClassList) {
	r.putRequiredTypes(imp, types)
}

func (r *PackageResolver) putObjectExtends(imp string, types *sppb.ClassList) {
	r.putRequiredTypes(imp, types)
}

func (r *PackageResolver) putRequiredTypes(imp string, types *sppb.ClassList) {
	for _, dst := range types.Classes {
		r.putRequiredType(imp, dst)
	}

}

// ResolveKnownImport implements the KnownImportResolver interface
func (r *PackageResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	if known, ok := r.registry.GetKnownImport(imp); ok {
		return known, nil
	}
	return r.next.ResolveKnownImport(c, ix, from, lang, imp)
}

func (r *PackageResolver) putRequiredType(src, dst string) {
	r.requiredTypes[dst] = append(r.requiredTypes[dst], src)
}

func (r *PackageResolver) Imports() ImportMap {
	m := NewImportMap()
	for imp, src := range r.requiredTypes {
		m.Put(NewExtendsImport(imp, src[0])) // use first occurrence as source arg
	}
	return m
}
