package resolver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/buildtools/build"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// TODO(pcj): rename this to 'Requirement' or something.  Not all Imports are
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

// NewSemanticImport creates a new semantic import from the given file.
func NewSemanticImport(imp string, source *sppb.File) *Import {
	return &Import{
		Kind:   sppb.ImportKind_SEMANTIC,
		Imp:    imp,
		Source: source,
	}
}

// NewResolvedNameImport creates a new resolved import from the given file,
// name, and symbol.  The 'name' is the token that resolved in the file scope.
func NewResolvedNameImport(imp string, source *sppb.File, name string, symbol *Symbol) *Import {
	return &Import{
		Kind:   sppb.ImportKind_RESOLVED_NAME,
		Imp:    imp,
		Source: source,
		Src:    name,
		Symbol: symbol,
	}
}

// NewTransitiveImport creates a new resolved import from the given file,
// name, and symbol.  The 'name' is the token that resolved in the file scope.
func NewTransitiveImport(imp string, name string, symbol *Symbol) *Import {
	return &Import{
		Kind:   sppb.ImportKind_TRANSITIVE,
		Imp:    imp,
		Src:    name,
		Symbol: symbol,
	}
}

// NewError creates a new err import from the given file,
// name, and symbol.
func NewErrorImport(imp string, source *sppb.File, src string, err error) *Import {
	return &Import{
		Kind:   sppb.ImportKind_IMPORT_KIND_UNKNOWN,
		Imp:    imp,
		Source: source,
		Src:    src,
		Error:  err,
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

	switch imp.Kind {
	case sppb.ImportKind_DIRECT:
		parts = append(parts, fmt.Sprintf("(%v of %s)", imp.Kind, filepath.Base(imp.Source.Filename)))
	case sppb.ImportKind_IMPLICIT:
		parts = append(parts, fmt.Sprintf("(%v via %q)", imp.Kind, imp.Src))
	case sppb.ImportKind_EXTENDS:
		parts = append(parts, fmt.Sprintf("(%v of %s via %q)", imp.Kind, filepath.Base(imp.Source.Filename), imp.Src))
	case sppb.ImportKind_RESOLVED_NAME:
		parts = append(parts, fmt.Sprintf("(%v of %s via %q)", imp.Kind, filepath.Base(imp.Source.Filename), imp.Src))
	case sppb.ImportKind_TRANSITIVE:
		parts = append(parts, fmt.Sprintf("(%v of %s)", imp.Kind, imp.Src))
	default:
		parts = append(parts, fmt.Sprintf("(%v)", imp.Kind))
	}
	return strings.Join(parts, " ")
}

func IsSelfImport(symbol *Symbol, repo, pkg, name string) bool {
	if symbol == nil {
		return false
	}
	if repo != "" {
		return false
	}
	if pkg != symbol.Label.Pkg {
		return false
	}
	if name != symbol.Label.Name {
		return false
	}
	return true
}

func IsWildcardImport(imp string) (string, bool) {
	if !strings.HasSuffix(imp, "._") {
		return "", false
	}
	return imp[:len(imp)-len("._")], true
}

func PutImportIfNotSelf(imports ImportMap, from label.Label) func(*Import) {
	return func(imp *Import) {
		if IsSelfImport(imp.Symbol, from.Repo, from.Pkg, from.Name) {
			return
		}
		imports.Put(imp)
	}
}
