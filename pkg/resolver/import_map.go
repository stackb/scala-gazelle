package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/buildtools/build"
)

// ImportLabel is a pair of (Import,Label)
type ImportLabel struct {
	Import *Import
	Label  label.Label
}

type ImportMap interface {
	Keys() []string
	Values() []*Import
	Deps(from label.Label) map[label.Label]*ImportLabel
	Put(imp *Import)
	Get(name string) (*Import, bool)
	Annotate(comments *build.Comments, accept func(imp *Import) bool)
}

// OrderedImportMap is a map if imports keyed by the import string.
type OrderedImportMap struct {
	values []*Import
	has    map[string]bool
}

// NewImportMap initializes a new ImportMap with an optional list of Imports.
func NewImportMap(imports ...*Import) ImportMap {
	m := &OrderedImportMap{
		has:    make(map[string]bool),
		values: make([]*Import, 0),
	}

	for _, imp := range imports {
		m.Put(imp)
	}

	return m
}

// Keys returns a sorted list of imports.
func (imports *OrderedImportMap) Keys() []string {
	keys := make([]string, len(imports.values))
	for i, imp := range imports.values {
		keys[i] = imp.Imp
	}
	return keys
}

// Values returns an ordered list of *Import reflecting the order in which it
// was added.
func (imports *OrderedImportMap) Values() []*Import {
	return imports.values
}

// Deps returns a de-duplicated list of labels that represent the from-relative
// final deps. The list is not sorted under the expectation that sorting will be
// done elsewhere.
func (imports *OrderedImportMap) Deps(from label.Label) map[label.Label]*ImportLabel {
	deps := make(map[label.Label]*ImportLabel)

	seen := map[label.Label]bool{
		label.NoLabel: true,
		from:          true,
	}

	for _, imp := range imports.values {
		if imp.Symbol == nil || imp.Error != nil {
			continue
		}
		dep := imp.Symbol.Label.Rel(from.Repo, from.Pkg)
		if seen[dep] {
			continue
		}
		seen[dep] = true
		// remove relative self imports (TODO: should these have been removed
		// earlier?)
		if dep.Relative && dep.Name == from.Name {
			continue
		}
		deps[dep] = &ImportLabel{Import: imp, Label: dep}
	}

	return deps
}

// Get the given import in the map, but only if it does not already exist in the map.
func (imports *OrderedImportMap) Get(key string) (*Import, bool) {
	for _, imp := range imports.values {
		if imp.Imp == key {
			return imp, true
		}
	}
	return nil, false
}

// Put the given import in the map, but only if it does not already exist in the map.
func (imports *OrderedImportMap) Put(imp *Import) {
	// TODO: should we record *all* imports for a given key?  Does priority matter?
	if imports.has[imp.Imp] {
		return
	}
	imports.has[imp.Imp] = true
	imports.values = append(imports.values, imp)
}

func (imports *OrderedImportMap) Annotate(comments *build.Comments, accept func(imp *Import) bool) {
	for _, imp := range imports.values {
		if !accept(imp) {
			continue
		}
		comments.Before = append(comments.Before, build.Comment{Token: "# " + imp.String()})
	}
}
