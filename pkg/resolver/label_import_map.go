package resolver

import (
	"github.com/bazelbuild/bazel-gazelle/label"
)

// LabelImportMap describes the imports provided be a label, and where each of
// those imports was derived from.
type LabelImportMap map[label.Label]ImportMap

// NewLabelImportMap creates a new map and pre-initializes the label.NoLabel
// slot with a non-nil ImportOriginMap
func NewLabelImportMap() LabelImportMap {
	resolved := make(LabelImportMap)
	resolved[label.NoLabel] = make(ImportMap)
	return resolved
}

func (m LabelImportMap) Set(from label.Label, imp *Import) {
	all, ok := m[from]
	if !ok {
		all = NewImportMap()
		m[from] = all
	}
	all[imp.Imp] = imp
}
