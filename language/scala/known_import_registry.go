package scala

import (
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// GetKnownImport implements part of the resolver.KnownImportRegistry interface.
func (sl *scalaLang) GetKnownImport(imp string) (*resolver.KnownImport, bool) {
	return sl.knownImports.GetKnownImport(imp)
}

// PutKnownImport implements part of the resolver.KnownImportRegistry interface.
func (sl *scalaLang) PutKnownImport(known *resolver.KnownImport) error {
	// log.Println("scalaLang.PutKnownImport", known)
	return sl.knownImports.PutKnownImport(known)
}
