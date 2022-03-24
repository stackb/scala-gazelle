package scala

import (
	"log"
	"sort"

	"github.com/RoaringBitmap/roaring"
)

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		symbols: make([]string, 0),
		indices: make(map[string]uint32),
	}
}

// SymbolTable provides an unsychronized set of strings.
type SymbolTable struct {
	symbols []string
	indices map[string]uint32
}

func (s *SymbolTable) Get(value string) (uint32, bool) {
	id, ok := s.indices[value]
	return id, ok
}

func (s *SymbolTable) Add(value string) uint32 {
	if index, ok := s.indices[value]; ok {
		return index
	} else {
		index := uint32(len(s.symbols))
		s.symbols = append(s.symbols, value)
		s.indices[value] = index
		return index
	}
}

func (s *SymbolTable) AddAll(values []string) BitSet {
	b := roaring.New()
	for _, value := range values {
		b.Add(s.Add(value))
	}
	return &roaringBitSet{b}
}

func (s *SymbolTable) Resolve(idx uint32) string {
	if int(idx) > len(s.symbols) {
		log.Panicf("index out of bounds: %d > %d", idx, len(s.symbols)-1)
	}
	return s.symbols[idx]
}

func (s *SymbolTable) ResolveAll(bits BitSet) (out []string) {
	it := bits.Iterator()
	for it.HasNext() {
		out = append(out, s.Resolve(it.Next()))
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
