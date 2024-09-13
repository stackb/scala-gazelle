package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/semanticdb"

	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

var (
	jarFile    string
	outputFile string
)

func main() {
	log.SetPrefix("semanticdbextract: ")
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

	err = parseFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %v", err)
	}

	doc, err := extract(jarFile)
	if err != nil {
		return fmt.Errorf("failed to merge files: %v", err)
	}

	if err := protobuf.WriteFile(outputFile, doc); err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

func parseFlags(args []string) (err error) {
	fs := flag.NewFlagSet("semanticdbextract", flag.ExitOnError)
	fs.StringVar(&jarFile, "jar_file", "", "the jar file to read")
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: semanticdbextract @PARAMS_FILE | semanticdbextract OPTIONS")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return err
	}

	if jarFile == "" {
		log.Fatal("-jar_file is required")
	}
	if outputFile == "" {
		log.Fatal("-output_file is required")
	}

	files := fs.Args()
	if len(files) != 0 {
		err = fmt.Errorf("semanticdbextract positional args should be empty")
	}

	return
}

func extract(filename string) (*spb.TextDocuments, error) {
	docs := new(spb.TextDocuments)
	seen := make(map[string]bool)

	addDocument := func(doc *spb.TextDocument) {
		if seen[doc.Uri] {
			return
		}
		seen[doc.Uri] = true
		docs.Documents = append(docs.Documents, doc)
	}

	group, err := semanticdb.ReadJarFile(filename)
	if err != nil {
		return nil, err
	}
	for _, docs := range group {
		for _, doc := range docs.Documents {
			addDocument(doc)
		}
	}

	sort.Slice(docs.Documents, func(i, j int) bool {
		a := docs.Documents[i].Uri
		b := docs.Documents[j].Uri
		return a < b
	})

	return docs, nil
}
