package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// KnownImport associates an import string with the label that provides it, along
// with a type classifier that says what kind of import this is.
type KnownImport struct {
	Type     sppb.ImportType
	Import   string
	Provider string
	Label    label.Label
}

func NewKnownImport(impType sppb.ImportType, imp, provider string, from label.Label) *KnownImport {
	return &KnownImport{
		Type:     impType,
		Import:   imp,
		Provider: provider,
		Label:    from,
	}
}

func (ki *KnownImport) String() string {
	return fmt.Sprintf("%v %s (%v)", ki.Type, ki.Import, ki.Label)
}
