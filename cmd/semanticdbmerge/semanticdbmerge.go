package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/semanticdb"

	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

var outputFile string

func main() {
	log.SetPrefix("semanticdbmerge: ")
	log.SetFlags(0) // don't print timestamps

	args, err := collections.ReadOsArgsParams()
	if err != nil {
		log.Fatalln("failed to read params file:", err)
	}

	files, err := parseFlags(args)
	if err != nil {
		log.Fatal(err)
	}

	merged, err := merge(files...)
	if err != nil {
		log.Fatal(err)
	}

	if err := protobuf.WriteFile(outputFile, merged); err != nil {
		log.Fatal(err)
	}
}

func parseFlags(args []string) (files []string, err error) {
	fs := flag.NewFlagSet("semanticdbmerge", flag.ExitOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: semanticdbmerge @PARAMS_FILE | semanticdbmerge OPTIONS FILES")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return
	}

	if outputFile == "" {
		log.Fatal("-output_file is required")
	}

	files = fs.Args()
	if len(files) == 0 {
		err = fmt.Errorf("semanticdbmerge positional args should be a non-empty list of scala jars (that contains semanticdb info) to merge")
	}

	return
}

func merge(filenames ...string) (*spb.TextDocuments, error) {
	seen := make(map[string]bool)
	merged := new(spb.TextDocuments)

	for _, filename := range filenames {
		group, err := semanticdb.ReadJarFile(filename)
		if err != nil {
			return nil, err
		}
		for _, item := range group {
			for _, doc := range item.Documents {
				if seen[doc.Uri] {
					log.Println("seen:", doc.Uri)
					continue
				}

				// remove occurrences and synthetics for file size as they are
				// not used
				doc.Occurrences = nil
				doc.Synthetics = nil

				merged.Documents = append(merged.Documents, doc)
			}
		}
	}

	sort.Slice(merged.Documents, func(i, j int) bool {
		a := merged.Documents[i].Uri
		b := merged.Documents[j].Uri
		return a < b
	})

	return merged, nil
}
