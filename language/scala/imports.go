package scala

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/label"
	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

func (sl *scalaLang) writeResolvedImportsMapFile(filename string) error {
	imports := &scpb.ResolvedImports{
		Imports: make(map[string]string),
	}

	for _, sym := range sl.globalScope.GetSymbols("") {
		dep := "NO_LABEL"
		if sym.Label != label.NoLabel {
			dep = sym.Label.String()
		}
		imports.Imports[sym.Name] = dep
	}

	if filepath.Ext(filename) == ".txt" {
		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("create: %w", err)
		}
		for k, v := range imports.Imports {
			fmt.Fprintln(f, k, v)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}
	} else {
		if err := protobuf.WriteFile(filename, imports); err != nil {
			return err
		}
	}

	log.Printf("Wrote scala-gazelle import map: %s", filename)

	return nil
}
