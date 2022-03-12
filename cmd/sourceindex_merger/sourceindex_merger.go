package main

import (
	"flag"
	"log"
	"os"

	"github.com/stackb/scala-gazelle/pkg/index"
)

const (
	debug = false
)

// outputFile holds the value of --output_file
var outputFile string

func main() {
	log.SetPrefix("sourceindex_merger: ")
	log.SetFlags(0) // don't print timestamps

	fs := flag.NewFlagSet("sourceindex_merger", flag.ContinueOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if outputFile == "" {
		log.Fatal("-output_file is required")
	}
	if len(fs.Args()) == 0 {
		log.Fatal("positional args should be a non-empty list of .sourceindex.json files to merge: args=", os.Args)
	}
	if debug {
		index.ListFiles(".")
	}
	if err := merge(fs.Args()); err != nil {
		log.Fatal(err)
	}
}

func merge(filenames []string) error {
	// idx is the final object to write as output
	var idx index.ScalaRuleIndexSpec

	// labels is used to prevent duplicate entries.
	labels := make(map[string]bool)

	for _, filename := range filenames {
		rule, err := index.ReadScalaRuleSpec(filename)
		if err != nil {
			return err
		}
		if labels[rule.Label] {
			if debug {
				log.Println("duplicate sourceindex spec:", rule.Label)
			}
			continue
		}
		labels[rule.Label] = true

		idx.Rules = append(idx.Rules, rule)
	}

	if err := index.WriteJSONFile(outputFile, idx); err != nil {
		return err
	}
	return nil
}
