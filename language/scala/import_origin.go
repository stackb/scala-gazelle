package scala

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/stackb/scala-gazelle/pkg/index"
)

const ImportKindDirect = ImportKind("direct")
const ImportKindTransitive = ImportKind("transitive")
const ImportKindSuperclass = ImportKind("superclass")
const ImportKindIndirect = ImportKind("indirect")
const ImportKindMainClass = ImportKind("main_class")
const ImportKindExport = ImportKind("export")
const ImportKindComment = ImportKind("comment")

type ImportKind string

// ImportOrigin is used to trace import provenance.
type ImportOrigin struct {
	Kind       ImportKind
	SourceFile *index.ScalaFileSpec
	Parent     string
	Children   []string // transitive imports triggered for an import
	Actual     string   // the effective string for the import.
}

func (io *ImportOrigin) String() string {
	var s string
	switch io.Kind {
	case ImportKindDirect:
		s = fmt.Sprintf("%s from %s", io.Kind, filepath.Base(io.SourceFile.Filename))
		if io.Parent != "" {
			s += " (materialized from " + io.Parent + ")"
		}
	case ImportKindExport:
		s = fmt.Sprintf("%s by %s", io.Kind, filepath.Base(io.SourceFile.Filename))
	case ImportKindIndirect:
		s = fmt.Sprintf("%s from %s", io.Kind, io.Parent)
	case ImportKindSuperclass:
		s = fmt.Sprintf("%s of %s", io.Kind, io.Parent)
	case ImportKindTransitive:
		s = fmt.Sprintf("%s via %s", io.Kind, io.Parent)
	case ImportKindComment:
		s = fmt.Sprintf("%s", io.Kind)
	default:
		return "unknown import origin kind: " + string(io.Kind)
	}
	if len(io.Children) > 0 {
		s += fmt.Sprintf(" (requires %v)", io.Children)
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
