package scala

import (
	"strings"

	"github.com/dghubble/trie"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func newKnownImportsTrie() *trie.PathTrie {
	return trie.NewPathTrieWithConfig(&trie.PathTrieConfig{
		Segmenter: importSegmenter,
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

// GetKnownImport implements part of the resolver.KnownImportRegistry interface.
func (sl *scalaLang) GetKnownImport(imp string) (*resolver.KnownImport, bool) {
	provider := sl.knownImports.Get(imp)
	if provider == nil {
		return nil, false
	}
	return provider.(*resolver.KnownImport), true
}

// PutKnownImport implements part of the resolver.KnownImportRegistry interface.
func (sl *scalaLang) PutKnownImport(known *resolver.KnownImport) error {
	sl.knownImports.Put(known.Import, known)
	return nil
}
