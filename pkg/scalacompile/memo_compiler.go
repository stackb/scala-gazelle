package scalacompile

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// MemoCompiler is a Compiler frontend that checks if the compilation time is
// zero.  If so, assume the compilation step has not occurred.
type MemoCompiler struct {
	next Compiler
}

func NewMemoCompiler(next Compiler) *MemoCompiler {
	return &MemoCompiler{
		next: next,
	}
}

// CompileScalaRule implements scalacompile.Compiler
func (p *MemoCompiler) CompileScalaRule(from label.Label, dir string, rule *sppb.Rule) error {
	if rule.CompileTimeMillis > 0 {
		return nil
	}
	return p.next.CompileScalaRule(from, dir, rule)
}
