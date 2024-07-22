package resolver

import (
	"fmt"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

const debugConflicts = false

// Symbol associates a name with the label that provides it, along with a type
// classifier that says what kind of symbol it is.
type Symbol struct {
	// Type is the kind of symbol this is.
	Type sppb.ImportType
	// Name is the fully-qualified import name.
	Name string
	// Label is the bazel label where the symbol is provided from.
	Label label.Label
	// Provider is the name of the provider that supplied the symbol.
	Provider string
	// Conflicts is a list of symbols provided by another provider or label.
	Conflicts []*Symbol
	// Requires is a list of other symbols that are required by this one.
	Requires []*Symbol
}

// NewSymbol constructs a new symbol pointer with the given arguments.
func NewSymbol(impType sppb.ImportType, name, provider string, from label.Label) *Symbol {
	return &Symbol{
		Type:     impType,
		Name:     name,
		Provider: provider,
		Label:    from,
	}
}

// Require adds a symbol to the requires list.
func (s *Symbol) Require(sym *Symbol) {
	s.Requires = append(s.Requires, sym)
}

// Conflict adds a symbol to the conflicts list.
func (s *Symbol) Conflict(sym *Symbol) {
	if debugConflicts {
		diff := cmp.Diff(s, sym, cmpopts.IgnoreFields(Symbol{}, "Conflicts"))
		if diff != "" {
			log.Printf("conflicting symbols %q: %s", s.Name, diff)
		}
	}
	s.Conflicts = append(s.Conflicts, sym)
}

// String implements fmt.Stringer
func (s *Symbol) String() string {
	return fmt.Sprintf("(%s<%v> %s<%v>)", s.Name, s.Type, s.Label, s.Provider)
}

func (s *Symbol) Proto() *sppb.Symbol {
	return &sppb.Symbol{
		Name: s.Name,
	}
}

func SymbolConfictMessage(symbol *Symbol, imp *Import, from label.Label) string {
	if len(symbol.Conflicts) == 0 {
		return ""
	}
	lines := make([]string, 0, len(symbol.Conflicts)+3)
	lines = append(lines, fmt.Sprintf("Ambiguous resolve of %v %q (symbol is provided by %d labels) [%s]", symbol.Type, symbol.Name, len(symbol.Conflicts)+1, imp))
	if symbol.Type == sppb.ImportType_PACKAGE || symbol.Type == sppb.ImportType_PROTO_PACKAGE {
		lines = append(lines, " - Possible action: remove wildcard or package import")
	}
	lines = append(lines, fmt.Sprintf(" - Possible action: add a resolve directive to %s:", label.Label{Repo: from.Repo, Pkg: from.Pkg, Name: "BUILD.bazel"}))
	for _, conflict := range append(symbol.Conflicts, symbol) {
		lines = append(lines, fmt.Sprintf("     # gazelle:resolve scala scala %s %s:", symbol.Name, conflict.Label))
	}
	return strings.Join(lines, "\n")
}
