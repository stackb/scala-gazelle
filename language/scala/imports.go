package scala

import (
	"log"

	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

func (sl *scalaLang) writeResolvedImportsMapFile(filename string) error {
	imports := &scpb.ResolvedImports{
		Imports: make(map[string]string),
	}

	for _, sym := range sl.globalScope.GetSymbols("") {
		imports.Imports[sym.Name] = sym.Label.String()
	}

	if err := protobuf.WriteFile(filename, imports); err != nil {
		return err
	}

	log.Printf("Wrote scala-gazelle import map %s", filename)

	return nil
}
