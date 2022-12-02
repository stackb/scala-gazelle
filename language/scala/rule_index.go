package scala

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

const debugLookupRule = false

// LookupRule implements part of the crossresolve.RuleIndex interface
func (sl *scalaLang) LookupRule(from label.Label) (*rule.Rule, bool) {
	r, ok := sl.allRules[from]
	if debugLookupRule {
		log.Printf("scalaLang.LookupRule(%q) -> %t", from, ok)
	}
	return r, ok
}

// recordRule sets the given rule in the global label->rule map.
func (sl *scalaLang) recordRule(from label.Label, r *rule.Rule) {
	if debugLookupRule {
		log.Printf("scalaLang.recordRule(%q) [%s]", from, r.Kind())
	}
	sl.allRules[from] = r
}

// LookupImport implements part of the crossresolve.RuleIndex interface
func (sl *scalaLang) LookupImport(imp resolve.ImportSpec) (*crossresolve.ImportProvider, bool) {
	provider := sl.allImports.Get(imp.Imp)
	if provider == nil {
		return nil, false
	}
	return provider.(*crossresolve.ImportProvider), true
}

// recordImport sets the given import in the global import trie.
func (sl *scalaLang) recordImport(imp resolve.ImportSpec, typ string, from label.Label) {
	sl.allImports.Put(imp.Imp, &crossresolve.ImportProvider{
		Label: from,
		Type:  typ,
	})
}

// importSegmenter segments string key paths by dot separators. For example,
// ".a.b.c" -> (".a", 2), (".b", 4), (".c", -1) in successive calls. It does
// not allocate any heap memory.
func importSegmenter(path string, start int) (segment string, next int) {
	if len(path) == 0 || start < 0 || start > len(path)-1 {
		return "", -1
	}
	end := strings.IndexRune(path[start+1:], '.') // next '.' after 0th rune
	if end == -1 {
		return path[start:], -1
	}
	return path[start : start+end+1], start + end + 1
}
