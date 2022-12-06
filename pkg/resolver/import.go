package resolver

import (
	"fmt"
	"path/filepath"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Import is used to trace import provenance.
type Import struct {
	// Kind is the import type
	Kind sppb.ImportKind
	// Imp is the name of the import
	Imp string
	// File is the source file (when this is a direct import).
	Source *sppb.File
	// Src is the name of the parent import (when this is an implicit import)
	Src string
	// Source is the resolved parent import (when this is an implicit import).
	Parent *Import
	// Known is the known provider of the import, after resolution.
	Known *KnownImport
	// Error is assiged if there is a resolution error.
	Error error
}

// NewDirectImport creates a new direct import from the given file.
func NewDirectImport(imp string, src *sppb.File) *Import {
	return &Import{
		Kind:   sppb.ImportKind_DIRECT,
		Imp:    imp,
		Source: src,
	}
}

// NewImplicitImport creates a new implicit import from the given parent src.
func NewImplicitImport(imp, src string) *Import {
	return &Import{
		Kind: sppb.ImportKind_IMPLICIT,
		Imp:  imp,
		Src:  src,
	}
}

// NewMainClassImport creates a new main_class import.
func NewMainClassImport(imp string) *Import {
	return &Import{
		Kind: sppb.ImportKind_MAIN_CLASS,
		Imp:  imp,
	}
}

func (imp *Import) String() string {
	s := imp.Imp + " ("
	switch imp.Kind {
	case sppb.ImportKind_DIRECT:
		if imp.Source == nil {
			panic("source file should always be set for direct import: this is a bug")
		}
		s += fmt.Sprintf("%v from %s", imp.Kind, filepath.Base(imp.Source.Filename))
	case sppb.ImportKind_IMPLICIT:
		if imp.Src == "" {
			panic("src/parent should always be set for an implicit import: this is a bug")
		}
		s += fmt.Sprintf("%v from %s", imp.Kind, imp.Parent)
	default:
		s += fmt.Sprintf("%v", imp.Kind)
	}
	s += ")"
	if imp.Known != nil {
		s += fmt.Sprintf(": resolved %v", imp.Known.Type)
		if imp.Known.Import != imp.Imp {
			s += "<" + imp.Known.Import + ">"
		}
		s += " to " + imp.Known.Label.String()
	}
	if imp.Error != nil {
		s += fmt.Sprintf(": error %v", imp.Error)
	}

	return s
}
