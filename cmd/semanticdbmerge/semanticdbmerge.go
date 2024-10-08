package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/semanticdb"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

var outputFile string

func main() {
	log.SetPrefix("semanticdbmerge: ")
	log.SetFlags(0) // don't print timestamps

	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	args, err := collections.ReadArgsParamsFile(args)
	if err != nil {
		return fmt.Errorf("failed to read params file: %v", err)
	}

	files, err := parseFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %v", err)
	}

	merged, err := merge(files...)
	if err != nil {
		return fmt.Errorf("failed to merge files: %v", err)
	}

	if err := protobuf.WriteFile(outputFile, merged); err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

func parseFlags(args []string) (files []string, err error) {
	fs := flag.NewFlagSet("semanticdbmerge", flag.ExitOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: semanticdbmerge @PARAMS_FILE | semanticdbmerge OPTIONS FILES")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return nil, err
	}

	if outputFile == "" {
		log.Fatal("-output_file is required")
	}

	files = fs.Args()
	if len(files) == 0 {
		err = fmt.Errorf("semanticdbmerge positional args should be a non-empty list of jars or semanticdb files to merge")
	}

	return
}

func merge(filenames ...string) (*sppb.FileSet, error) {
	fileSet := new(sppb.FileSet)
	seen := make(map[string]bool)

	addFile := func(file *sppb.File) {
		if seen[file.Filename] {
			return
		}
		seen[file.Filename] = true
		fileSet.Files = append(fileSet.Files, file)
	}

	addDocument := func(doc *spb.TextDocument) {
		if seen[doc.Uri] {
			return
		}
		addFile(newFile(doc))
	}

	for _, filename := range filenames {
		switch filepath.Ext(filename) {
		case ".pb":
			var fileSet sppb.FileSet
			err := protobuf.ReadFile(filename, &fileSet)
			if err != nil {
				return nil, err
			}
			for _, file := range fileSet.Files {
				addFile(file)
			}
		case ".jar":
			group, err := semanticdb.ReadJarFile(filename)
			if err != nil {
				return nil, err
			}
			for _, docs := range group {
				for _, doc := range docs.Documents {
					addDocument(doc)
				}
			}
		}
	}

	sort.Slice(fileSet.Files, func(i, j int) bool {
		a := fileSet.Files[i].Filename
		b := fileSet.Files[j].Filename
		return a < b
	})

	return fileSet, nil
}

func newFile(doc *spb.TextDocument) *sppb.File {
	return &sppb.File{
		Filename:        doc.Uri,
		SemanticImports: semanticdb.SemanticImports(doc),
	}
}
