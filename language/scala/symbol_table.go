package scala

import (
	"log"
	"sort"
	"strings"

	"github.com/RoaringBitmap/roaring"
)

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		symbols: make([]string, 0),
		ids:     make(map[string]uint32),
	}
}

// SymbolTable provides an unsychronized set of strings.
type SymbolTable struct {
	symbols []string
	ids     map[string]uint32
}

func (s *SymbolTable) Get(value string) (uint32, bool) {
	id, ok := s.ids[value]
	return id, ok
}

func (s *SymbolTable) Add(value string) uint32 {
	id, ok := s.ids[value]
	if ok {
		return id
	}
	id = uint32(len(s.symbols))
	s.symbols = append(s.symbols, value)
	s.ids[value] = id

	if got := s.resolveAt(id); got != value {
		log.Panicf("dep %q (id=%d) was not idempotent to add (got %q instead)", value, id, got)
	}

	// log.Printf("symbolTable: added %q (id=%d)", value, id)

	return id
}

func (s *SymbolTable) AddAll(values []string) BitSet {
	b := roaring.New()
	for _, value := range values {
		b.Add(s.Add(value))
	}
	return &roaringBitSet{b}
}

func (s *SymbolTable) resolveAt(idx uint32) string {
	if int(idx) >= len(s.symbols) {
		log.Panicf("index out of bounds: %d > %d", idx, len(s.symbols)-1)
	}
	return s.symbols[idx]
}

func (s *SymbolTable) ResolveAll(bits BitSet, kind string) (out []string) {
	prefix := kind + "/"
	it := bits.Iterator()
	for it.HasNext() {
		got := s.resolveAt(it.Next())
		if !strings.HasPrefix(got, prefix) {
			continue
		}
		out = append(out, strings.TrimPrefix(got, prefix))
	}
	sort.Strings(out)
	return
}

type BitSet interface {
	Iterator() Iterator
}

type roaringBitSet struct {
	*roaring.Bitmap
}

func (b *roaringBitSet) Iterator() Iterator {
	return &roaringIterator{b.Bitmap.Iterator()}
}

type Iterator interface {
	HasNext() bool
	Next() uint32
}

type roaringIterator struct {
	roaring.IntPeekable
}

func (i *roaringIterator) HasNext() bool {
	return i.IntPeekable.HasNext()
}

func (i *roaringIterator) Next() uint32 {
	return i.IntPeekable.Next()
}
