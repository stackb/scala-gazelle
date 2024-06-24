package scala

import (
	"fmt"
	"os"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GetKnownFile implements part of the resolver.KnownFileRegistry interface.
func (sl *scalaLang) GetKnownFile(pkg string) (*rule.File, bool) {
	r, ok := sl.knownFiles[pkg]
	return r, ok
}

// PutKnownFile implements part of the resolver.KnownFileRegistry interface.
func (sl *scalaLang) PutKnownFile(pkg string, r *rule.File) error {
	if _, ok := sl.knownFiles[pkg]; ok {
		return fmt.Errorf("duplicate known file: %s", pkg)
	}
	sl.knownFiles[pkg] = r
	return nil
}

func (sl *scalaLang) emitKnownFiles() error {
	for pkg, file := range sl.knownFiles {
		if err := sl.emitKnownFile(pkg, file); err != nil {
			return err
		}
	}
	if false {
		return fmt.Errorf("known files: WIP")
	}
	return nil
}

func (sl *scalaLang) emitKnownFile(_ string, file *rule.File) error {
	if err := os.WriteFile(file.Path, file.Format(), 0o666); err != nil {
		return err
	}
	return nil
}
