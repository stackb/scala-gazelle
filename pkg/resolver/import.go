package resolver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
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

// NewExtendsImport creates a new extends import from the given requiring type.
func NewExtendsImport(imp, src string) *Import {
	return &Import{
		Kind: sppb.ImportKind_EXTENDS,
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

func (imp *Import) Comment() build.Comment {
	return build.Comment{Token: "# " + imp.String()}
}

func (imp *Import) String() string {
	var impType string
	if imp.Known != nil {
		impType = fmt.Sprintf("%v", imp.Known.Type)
	} else if imp.Error != nil {
		impType = "ERROR"
	}
	parts := []string{
		fmt.Sprintf("%s<%s>", imp.Imp, impType),
	}

	if imp.Known != nil {
		to := imp.Known.Label.String()
		if to == "//:" {
			to = "NO-LABEL"
		}
		parts = append(parts, fmt.Sprintf("✅ %s<%s>", to, imp.Known.Provider))
	} else if imp.Error != nil {
		parts = append(parts, fmt.Sprintf("❌ %v", imp.Error))
	}

	if imp.Source != nil {
		parts = append(parts, fmt.Sprintf("(%v of %s)", imp.Kind, filepath.Base(imp.Source.Filename)))
	} else if imp.Src != "" {
		parts = append(parts, fmt.Sprintf("(%v of %s)", imp.Kind, imp.Src))
	} else {
		parts = append(parts, fmt.Sprintf("(%v)", imp.Kind))
	}
	return strings.Join(parts, " ")
}
