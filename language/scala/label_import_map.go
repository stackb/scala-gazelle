package scala

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

// LabelImportMap describes the imports provided be a label, and where each of
// those imports was derived from.
type LabelImportMap map[label.Label]ImportOriginMap

// NewLabelImportMap creates a new map and pre-initializes the label.NoLabel
// slot with a non-nil ImportOriginMap
func NewLabelImportMap() LabelImportMap {
	resolved := make(LabelImportMap)
	resolved[label.NoLabel] = make(ImportOriginMap)
	return resolved
}

func (m LabelImportMap) Set(from label.Label, imp string, origin *ImportOrigin) {

	if all, ok := m[from]; ok {
		all[imp] = origin
	} else {
		m[from] = map[string]*ImportOrigin{imp: origin}
	}
	// if debug {
	// 	log.Printf(" --> resolved %q (%s) to %v", imp, origin.String(), from)
	// }
}

func (m LabelImportMap) String() string {
	var sb strings.Builder
	for from, imports := range m {
		sb.WriteString(from.String())
		sb.WriteString(":\n")
		for imp, origin := range imports {
			sb.WriteString(" -- ")
			sb.WriteString(imp)
			sb.WriteString(" -> ")
			sb.WriteString(origin.String())
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
