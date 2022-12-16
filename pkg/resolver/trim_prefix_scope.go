package resolver

import (
	"fmt"
	"strings"
)

// TrimPrefixScope implements Scope that trims a prefix from the lookup name.
type TrimPrefixScope struct {
	prefix string
	next   Scope
}

func NewTrimPrefixScope(prefix string, next Scope) *TrimPrefixScope {
	return &TrimPrefixScope{
		prefix: prefix,
		next:   next,
	}
}

// PutSymbol is not supported and will panic.
func (r *TrimPrefixScope) PutSymbol(known *Symbol) error {
	return fmt.Errorf("unsupported operation: PutSymbol")
}

// GetSymbol implements part of the Scope interface
func (r *TrimPrefixScope) GetSymbol(name string) (*Symbol, bool) {
	return r.next.GetSymbol(strings.TrimPrefix(name, r.prefix))
}

// GetScope implements part of the resolver.Scope interface.
func (r *TrimPrefixScope) GetScope(name string) (Scope, bool) {
	return r.next.GetScope(strings.TrimPrefix(name, r.prefix))
}

// GetSymbols implements part of the Scope interface
func (r *TrimPrefixScope) GetSymbols(name string) []*Symbol {
	return r.next.GetSymbols(strings.TrimPrefix(name, r.prefix))
}
