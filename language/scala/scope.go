package scala

import "github.com/stackb/scala-gazelle/pkg/resolver"

// GetSymbol implements part of the resolver.Scope interface.
func (sl *scalaLang) GetSymbol(imp string) (*resolver.Symbol, bool) {
	return sl.knownImports.GetSymbol(imp)
}

// GetSymbols implements part of the resolver.Scope interface.
func (sl *scalaLang) GetSymbols(prefix string) []*resolver.Symbol {
	return sl.knownImports.GetSymbols(prefix)
}

// PutSymbol implements part of the resolver.Scope interface.
func (sl *scalaLang) PutSymbol(known *resolver.Symbol) error {
	// log.Println("scalaLang.PutSymbol", known)
	return sl.knownImports.PutSymbol(known)
}
