package resolver

import (
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/buildtools/build"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// ImportMap is a map if imports keyed by the import string.
type ImportMap map[string]*Import

// NewImportMap initializes a new ImportMap with an optional list of Imports.
func NewImportMap(imports ...*Import) ImportMap {
	m := make(ImportMap)
	for _, imp := range imports {
		m.Put(imp)
	}
	return m
}

// Keys returns a sorted list of imports.
func (imports ImportMap) Keys() []string {
	keys := make([]string, len(imports))
	i := 0
	for k := range imports {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// Values returns a sorted list of *Import.
func (imports ImportMap) Values() []*Import {
	vals := make([]*Import, len(imports))
	for i, k := range imports.Keys() {
		vals[i] = imports[k]
	}
	return vals
}

// Deps returns a de-duplicated list of labels that represent the from-relative
// final deps. The list is not sorted under the expectation that sorting will be
// done elsewhere.
func (imports ImportMap) Deps(from label.Label) (deps []label.Label) {
	seen := map[label.Label]bool{
		label.NoLabel: true,
		from:          true,
	}

	for _, k := range imports.Keys() {
		imp := imports[k]
		if imp.Symbol == nil || imp.Error != nil {
			continue
		}
		dep := imp.Symbol.Label
		if seen[dep] {
			continue
		}
		seen[dep] = true
		relDep := dep.Rel(from.Repo, from.Pkg)
		// remove relative self imports (TODO: should these have been removed
		// earlier?)
		if relDep.Relative && relDep.Name == from.Name {
			continue
		}
		deps = append(deps, relDep)
	}
	return
}

// Has checks if the given import key is already present in the map.
func (imports ImportMap) Has(imp string) bool {
	_, ok := imports[imp]
	return ok
}

// Put the given import in the map.
func (imports ImportMap) Put(imp *Import) {
	// TODO: should we record *all* imports for a given key?  Does priority matter?
	if _, ok := imports[imp.Imp]; !ok {
		imports[imp.Imp] = imp
	}
}

func (imports ImportMap) Annotate(comments *build.Comments, accept func(imp *Import) bool) {
	for _, key := range imports.Keys() {
		imp := imports[key]
		if !accept(imp) {
			continue
		}
		comments.Before = append(comments.Before, build.Comment{Token: "# " + imp.String()})
	}
}

func (imports ImportMap) ProtoList() []*sppb.Import {
	list := make([]*sppb.Import, len(imports))
	for i, imp := range imports.Values() {
		list[i] = imp.Proto()
	}
	return list
}
