package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// Symbol associates a name with the label that provides it, along with a type
// classifier that says what kind of symbol it is.
type Symbol struct {
	Type     sppb.ImportType
	Name     string
	Label    label.Label
	Provider string
}

func NewSymbol(impType sppb.ImportType, name, provider string, from label.Label) *Symbol {
	return &Symbol{
		Type:     impType,
		Name:     name,
		Provider: provider,
		Label:    from,
	}
}

func (s *Symbol) String() string {
	return fmt.Sprintf("%v %s (%v)", s.Type, s.Name, s.Label)
}
