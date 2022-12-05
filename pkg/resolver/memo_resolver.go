package resolver

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// MemoResolver implements KnownImportResolver, memoizing results.
type MemoResolver struct {
	next  KnownImportResolver
	known map[string]*KnownImport
}

func NewMemoResolver(next KnownImportResolver) *MemoResolver {
	return &MemoResolver{
		next:  next,
		known: make(map[string]*KnownImport),
	}
}

// ResolveKnownImport implements the KnownImportResolver interface
func (r *MemoResolver) ResolveKnownImport(c *config.Config, ix *resolve.RuleIndex, from label.Label, lang string, imp string) (*KnownImport, error) {
	// log.Printf("memo.ResolveKnownImport(%q)", imp)
	if known, ok := r.known[imp]; ok {
		return known, nil
	}
	known, err := r.next.ResolveKnownImport(c, ix, from, lang, imp)
	if known != nil && err == nil {
		log.Printf("memo.ResolveKnownImport(%q) -> %s", imp, known)
		r.known[imp] = known
	}
	return known, err
}
