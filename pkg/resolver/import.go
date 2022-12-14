package resolver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// TODO: rename this to 'Requirement' or something.  Not all Imports are
// actually from an import statement.

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
	// Symbol is the resolved symbol of the import, or nil if not resolved.
	Symbol *Symbol
	// Error is assiged if there is a resolution error.
	Error error
}

// NewDirectImport creates a new direct import from the given file.
func NewDirectImport(imp string, source *sppb.File) *Import {
	return &Import{
		Kind:   sppb.ImportKind_DIRECT,
		Imp:    imp,
		Source: source,
	}
}

// NewResolvedSymbolImport creates a new resolved import from the given file,
// name, and symbol.  The 'name' is the token that resolved in the file scope.
func NewResolvedSymbolImport(imp string, source *sppb.File, name string, symbol *Symbol) *Import {
	return &Import{
		Kind:   sppb.ImportKind_RESOLVED_SYMBOL,
		Imp:    imp,
		Source: source,
		Src:    name,
		Symbol: symbol,
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
func NewExtendsImport(imp string, source *sppb.File, src string, symbol *Symbol) *Import {
	return &Import{
		Kind:   sppb.ImportKind_EXTENDS,
		Imp:    imp,
		Source: source,
		Src:    src,
		Symbol: symbol,
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
	emoji := "✅"

	if imp.Symbol != nil {
		impType = fmt.Sprintf("%v", imp.Symbol.Type)
	} else if imp.Error != nil {
		impType = "ERROR"
		emoji = "❌"
	}
	parts := []string{
		fmt.Sprintf("%s %s<%s>", emoji, imp.Imp, impType),
	}

	if imp.Symbol != nil {
		to := imp.Symbol.Label.String()
		if to == "//:" {
			to = "NO-LABEL"
		}
		parts = append(parts, fmt.Sprintf("%s<%s>", to, imp.Symbol.Provider))
	} else if imp.Error != nil {
		parts = append(parts, fmt.Sprintf("%v", imp.Error))
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
