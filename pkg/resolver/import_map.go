package resolver

import (
	"sort"

	"github.com/bazelbuild/buildtools/build"
)

// ImportMap is a map if imports keyed by the import string.
type ImportMap map[string]*Import

func NewImportMap() ImportMap {
	return make(ImportMap)
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

// Put the given import in the map.
func (imports ImportMap) Put(imp *Import) {
	// TODO: should we record *all* imports for a given key?
	imports[imp.Imp] = imp
}

func (imports ImportMap) HasErrors() bool {
	for _, imp := range imports {
		if imp.Error != nil {
			return true
		}
	}
	return false
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
