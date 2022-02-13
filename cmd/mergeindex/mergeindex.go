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

var outputFile string

func main() {
	log.SetPrefix("mergeindex: ")
	log.SetFlags(0) // don't print timestamps

	fs := flag.NewFlagSet("mergeindex", flag.ContinueOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if outputFile == "" {
		log.Fatal("-output_file is required")
	}
	if len(fs.Args()) == 0 {
		log.Fatal("positional args should be a non-empty list of .jarindex.json files to merge: args=", os.Args)
	}
	if debug {
		index.ListFiles(".")
	}
	if err := merge(fs.Args()); err != nil {
		log.Fatal(err)
	}
}

func merge(filenames []string) error {
	// spec is the final object to write as output
	var spec index.IndexSpec

	// labelByClass is used to check if more than one label provides a given
	// class.
	labelByClass := make(map[string][]string)

	for _, filename := range filenames {
		jarSpec, err := index.ReadJarSpec(filename)
		if err != nil {
			return err
		}
		for _, class := range jarSpec.Classes {
			labelByClass[class] = append(labelByClass[class], jarSpec.Label)
		}
		spec.JarSpecs = append(spec.JarSpecs, *jarSpec)
	}

	for classname, labels := range labelByClass {
		if len(labels) > 1 {
			log.Printf("class is provided by more than one label: %s: %v", classname, labels)
		}
	}

	if err := index.WriteJSONFile(outputFile, spec); err != nil {
		return err
	}
	return nil
}
