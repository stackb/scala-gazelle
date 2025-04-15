package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/jarindex"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

const debug = false

var (
	outputFile     string
	predefinedDeps collections.StringSlice
	preferredDeps  collections.StringSlice
	preferred      map[string]string
)

func main() {
	log.SetPrefix("mergeindex: ")
	log.SetFlags(0) // don't print timestamps

	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	args, err := collections.ReadArgsParamsFile(args)
	if err != nil {
		return err
	}

	files, err := parseFlags(args)
	if err != nil {
		return err
	}

	index, err := merge(files...)
	if err != nil {
		return err
	}

	if err := protobuf.WriteFile(outputFile, index); err != nil {
		return err
	}

	return nil
}

func parseFlags(args []string) (files []string, err error) {
	fs := flag.NewFlagSet("mergeindex", flag.ExitOnError)
	fs.Var(&preferredDeps, "preferred", "a repeatable list of mappings of the form PKG=DEP that declares which dependency should be chosen to resolve package ambiguity")
	fs.Var(&predefinedDeps, "predefined", "a repeatable list of labels to be considered predefined")
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: mergeindex @PARAMS_FILE | mergeindex OPTIONS FILES")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return
	}

	if outputFile == "" {
		log.Fatal("-output_file is required")
	}

	preferred = make(map[string]string)
	for _, v := range preferredDeps {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed --preferred argument, wanted PACKAGE=DEP (got %s)", v)
		}
		preferred[parts[0]] = parts[1]
	}

	files = fs.Args()
	if len(files) == 0 {
		err = fmt.Errorf("mergeindex positional args should be a non-empty list of jarindex.{pb|json} files to merge")
	}

	return
}

func merge(filenames ...string) (*jipb.JarIndex, error) {
	jars := make([]*jipb.JarFile, 0, len(filenames))
	for _, filename := range filenames {
		idx := jipb.JarIndex{}
		if err := protobuf.ReadFile(filename, &idx); err != nil {
			return nil, fmt.Errorf("reading jarindex file %q: %w", filename, err)
		}
		jars = append(jars, idx.JarFile...)
		if debug {
			if err := writeJarIndexJarFileJSONFiles(&idx); err != nil {
				return nil, err
			}
		}
	}

	index, err := jarindex.MergeJarFiles(func(format string, args ...interface{}) {
		log.Printf("warning: "+format, args...)
	}, predefinedDeps, jars)
	if err != nil {
		return nil, err
	}

	index.Predefined = predefinedDeps
	index.Preferred = preferred

	return index, nil
}

func writeJarIndexJarFileJSONFiles(idx *jipb.JarIndex) error {
	for _, file := range idx.JarFile {
		if err := writeJarFileJSONFile(file); err != nil {
			return err
		}
	}
	return nil
}

func writeJarFileJSONFile(file *jipb.JarFile) error {
	jarFilename := "/tmp/" + filepath.Base(file.Filename) + ".json"
	if err := protobuf.WriteFile(jarFilename, file); err != nil {
		return err
	}
	log.Println("Wrote:", jarFilename)
	return nil
}
