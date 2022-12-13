package resolver

import (
	"log"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

var scopePathTrieConfig = &collections.PathTrieConfig{
	Segmenter: importSegmenter,
}

// TrieScope implements Scope using a trie.
type TrieScope struct {
	trie *collections.PathTrie
}

// TrieScope constructs a new TrieScope.
func NewTrieScope() *TrieScope {
	return &TrieScope{
		trie: collections.NewPathTrieWithConfig(scopePathTrieConfig),
	}
}

// GetScope implements part of the resolver.Scope interface.
func (r *TrieScope) GetScope(name string) (Scope, bool) {
	node := r.trie.Get(name)
	if node == nil {
		return nil, false
	}
	return &TrieScope{node}, true
}

// GetSymbols implements part of the resolver.Scope interface.
func (r *TrieScope) GetSymbols(prefix string) (symbols []*Symbol) {
	node := r.trie.Get(prefix)
	if node == nil {
		return
	}
	node.Walk(func(key string, value interface{}) error {
		symbols = append(symbols, value.(*Symbol))
		return nil
	})
	sort.Slice(symbols, func(i, j int) bool {
		a := symbols[i]
		b := symbols[j]
		return a.Name < b.Name
	})
	return
}

// GetSymbol implements part of the resolver.Scope interface.
func (r *TrieScope) GetSymbol(imp string) (*Symbol, bool) {
	var last interface{}
	r.trie.WalkPath(imp, func(key string, value interface{}) error {
		last = value
		return nil
	})
	if last == nil {
		return nil, false
	}
	return last.(*Symbol), true
}

// PutSymbol implements part of the Scope interface.
func (r *TrieScope) PutSymbol(symbol *Symbol) error {
	if symbol.Provider == "" {
		log.Panicf("fatal (missing provider): %+v", symbol)
	}
	if current, ok := r.GetSymbol(symbol.Name); ok && current.Name == symbol.Name {
		if current.Label != symbol.Label {
			current.Conflicts = append(current.Conflicts, symbol)
			return nil
		}
		if false {
			diff := cmp.Diff(current, symbol, cmpopts.IgnoreFields(Symbol{}, "Conflicts"))
			if diff != "" {
				// log.Printf("conflicting symbols %q: %s", current.Name, diff)
				return nil
			}
		}
	}
	r.trie.Put(symbol.Name, symbol)
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
