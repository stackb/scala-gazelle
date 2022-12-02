package scala

import (
	"fmt"
	"path/filepath"
	"sort"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

const ImportKindDirect = ImportKind("direct")
const ImportKindImplicit = ImportKind("implicit")
const ImportKindMainClass = ImportKind("main_class")
const ImportKindComment = ImportKind("comment")

type ImportKind string

// ImportOrigin is used to trace import provenance.
type ImportOrigin struct {
	Kind       ImportKind
	SourceFile *sppb.File
	Parent     string
	Children   []string // transitive imports triggered for an import
	// Import holds the symbol that resolved.  For example, the string "com.foo" when .Actual is "com.foo._"
	Import string
	// Actual holds the original symbol.  For example "com.foo._".
	Actual string // the effective string for the import.
}

func NewDirectImportOrigin(src *sppb.File) *ImportOrigin {
	return &ImportOrigin{
		Kind:       ImportKindDirect,
		SourceFile: src,
	}
}

func NewImplicitImportOrigin(parent string) *ImportOrigin {
	return &ImportOrigin{
		Kind:   ImportKindImplicit,
		Parent: parent,
	}
}

func (origin *ImportOrigin) String() string {
	var s string
	switch origin.Kind {
	case ImportKindDirect:
		if origin.SourceFile == nil {
			panic("source file should always be set for direct import: this is a bug")
		}
		s += fmt.Sprintf("%s from %s", origin.Kind, filepath.Base(origin.SourceFile.Filename))
		if origin.Parent != "" {
			s += " (materialized from " + origin.Parent + ")"
		}
	case ImportKindImplicit:
		s += fmt.Sprintf("%s from %s", origin.Kind, origin.Parent)
	case ImportKindMainClass:
		s += fmt.Sprintf("%s", origin.Kind)
	case ImportKindComment:
		s += fmt.Sprintf("%s", origin.Kind)
	default:
		return "unknown import origin kind: " + string(origin.Kind)
	}

	if len(origin.Children) > 0 {
		s += fmt.Sprintf(" (requires %v)", origin.Children)
	}

	return s
}

type ImportOriginMap map[string]*ImportOrigin

func (m ImportOriginMap) Keys() []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func (m ImportOriginMap) Add(imp string, origin *ImportOrigin) {
	origin.Import = imp
	m[imp] = origin
}
