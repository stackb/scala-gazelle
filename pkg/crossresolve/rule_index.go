package crossresolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// RuleIndex is an index of known rules indexed by their label.
type RuleIndex interface {
	// LookupRule is a function that returns the generated rule for the given label
	LookupRule(from label.Label) (*rule.Rule, bool)
	// LookupImport is a function that returns the providing rule label for
	// the given import prefix.
	LookupImport(imp resolve.ImportSpec) (provider *ImportProvider, ok bool)
}

type ImportProvider struct {
	Label label.Label
	Type  sppb.ImportType
}
