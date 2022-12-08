package resolver

import (
	"log"
	"sort"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/collections"
)

// KnownImportRegistryTrie implements KnownImportRegistry using a trie.
type KnownImportRegistryTrie struct {
	known *collections.PathTrie
}

// KnownImportRegistryTrie constructs a new KnownImportRegistryTrie.
func NewKnownImportRegistryTrie() *KnownImportRegistryTrie {
	return &KnownImportRegistryTrie{
		known: collections.NewPathTrieWithConfig(&collections.PathTrieConfig{
			Segmenter: importSegmenter,
		}),
	}
}

// GetKnownImports implements part of the resolver.KnownImportRegistry interface.
func (r *KnownImportRegistryTrie) GetKnownImports(prefix string) (known []*KnownImport) {
	node := r.known.Get(prefix)
	if node == nil {
		return
	}
	node.Walk(func(key string, value interface{}) error {
		known = append(known, value.(*KnownImport))
		return nil
	})
	sort.Slice(known, func(i, j int) bool {
		a := known[i]
		b := known[j]
		return a.Import < b.Import
	})
	return
}

// GetKnownImport implements part of the resolver.KnownImportRegistry interface.
func (r *KnownImportRegistryTrie) GetKnownImport(imp string) (*KnownImport, bool) {
	var last interface{}
	r.known.WalkPath(imp, func(key string, value interface{}) error {
		last = value
		return nil
	})
	if last == nil {
		return nil, false
	}
	return last.(*KnownImport), true
}

// PutKnownImport implements part of the KnownImportRegistry interface.
func (r *KnownImportRegistryTrie) PutKnownImport(known *KnownImport) error {
	if known.Provider == "" {
		log.Panicf("fatal (missing provider): %+v", known)
	}
	r.known.Put(known.Import, known)
	return nil
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
