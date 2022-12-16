package scala

import "github.com/stackb/scala-gazelle/pkg/resolver"

// GetScope implements part of the resolver.Scope interface.
func (sl *scalaLang) GetScope(imp string) (resolver.Scope, bool) {
	return sl.globalScope.GetScope(imp)
}

// GetSymbol implements part of the resolver.Scope interface.
func (sl *scalaLang) GetSymbol(imp string) (*resolver.Symbol, bool) {
	return sl.globalScope.GetSymbol(imp)
}

// GetSymbols implements part of the resolver.Scope interface.
func (sl *scalaLang) GetSymbols(prefix string) []*resolver.Symbol {
	return sl.globalScope.GetSymbols(prefix)
}

// PutSymbol implements part of the resolver.Scope interface.
func (sl *scalaLang) PutSymbol(symbol *resolver.Symbol) error {
	return sl.globalScope.PutSymbol(symbol)
}

// String implements the fmt.Stringer interface.
func (sl *scalaLang) String() string {
	return sl.globalScope.String()
}
