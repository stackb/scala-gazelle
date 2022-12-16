package resolver

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

// LabelNameRewriteSpec is a specification to rewrite a label name.  For
// example, a bazel macro like `custom_scala_library` might be implemented such
// that the macro instantiates a "real" scala_library rule as `name + "_lib"`.
// In this case, we don't want to resolve the actual macro name but rather the
// lib name.  Given this example, it would be
// `LabelNameRewriteSpec{Src:"%{name}", Dst:"%{name}_lib"}`, where `%{name}` is
// a special token used as a placeholder for the label.Name.
type LabelNameRewriteSpec struct {
	// Src is the label name pattern to match
	Src string
	// Dst is the label name pattern to rewrite
	Dst string
}

func (m *LabelNameRewriteSpec) Rewrite(from label.Label) label.Label {
	if !(m.Src == from.Name || m.Src == "%{name}") {
		return from
	}
	return label.Label{Repo: from.Repo, Pkg: from.Pkg, Name: strings.ReplaceAll(m.Dst, "%{name}", from.Name)}
}
