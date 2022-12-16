package resolver

import (
	"fmt"
	"strings"
)

// ChainScope implements Scope over a chain of registries.
type ChainScope struct {
	chain []Scope
}

func NewChainScope(chain ...Scope) *ChainScope {
	return &ChainScope{
		chain: chain,
	}
}

// PutSymbol is not supported and will panic.
func (r *ChainScope) PutSymbol(known *Symbol) error {
	return fmt.Errorf("unsupported operation: PutSymbol")
}

// GetSymbol implements part of the Scope interface
func (r *ChainScope) GetSymbol(imp string) (*Symbol, bool) {
	for _, next := range r.chain {
		if known, ok := next.GetSymbol(imp); ok {
			return known, true
		}
	}
	return nil, false
}

// GetScope implements part of the resolver.Scope interface.
func (r *ChainScope) GetScope(imp string) (Scope, bool) {
	for _, next := range r.chain {
		if scope, ok := next.GetScope(imp); ok {
			return scope, true
		}
	}
	return nil, false
}

// GetSymbols implements part of the Scope interface
func (r *ChainScope) GetSymbols(prefix string) []*Symbol {
	for _, next := range r.chain {
		if known := next.GetSymbols(prefix); len(known) > 0 {
			return known
		}
	}
	return nil
}

// String implements the fmt.Stringer interface
func (r *ChainScope) String() string {
	var buf strings.Builder
	for _, next := range r.chain {
		buf.WriteString(next.String())
		buf.WriteRune('\n')
	}
	return buf.String()
}
