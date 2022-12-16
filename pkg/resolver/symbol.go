package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

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

// String implements fmt.Stringer
func (s *Symbol) String() string {
	return fmt.Sprintf("(%s<%v> %s<%v>)", s.Name, s.Type, s.Label, s.Provider)
}
