package resolver

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// KnownImport associates an import string with the label that provides it, along
// with a type classifier that says what kind of import this is.
type KnownImport struct {
	Import string
	Label  label.Label
	Type   sppb.ImportType
}

func (ki *KnownImport) String() string {
	return fmt.Sprintf("%v (from %v)", ki.Type, ki.Label)
}