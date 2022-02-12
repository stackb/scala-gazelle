package main

import (
	"archive/zip"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/index"
	"github.com/stackb/scala-gazelle/pkg/java"
)

const (
	debug = false
)

var (
	inputFile  string
	outputFile string
)

func main() {
	log.SetPrefix("jarindexer: ")
	log.SetFlags(0) // don't print timestamps

	fs := flag.NewFlagSet("jarindexer", flag.ContinueOnError)
	fs.StringVar(&inputFile, "input_file", "", "the input configuration file")
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if inputFile == "" {
		log.Fatal("-input_file is required")
	}
	if inputFile == "" {
		log.Fatal("-output_file is required")
	}
	if debug {
		index.ListFiles(".")
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	spec, err := index.ReadJarSpec(inputFile)
	if err != nil {
		return err
	}
	if err := parseJarFile(spec.Filename, spec); err != nil {
		log.Printf("warning: could not parse %s: %v", spec.Filename, err)
	}
	if err := index.WriteJarSpec(outputFile, spec); err != nil {
		return err
	}
	return nil
}

func parseJarFile(filename string, spec *index.JarSpec) error {
	log.Println("Parsing jar file:", filename)
	entry := java.NewJarClassPathEntry(filename)
	return entry.Visit(func(f *zip.File, c *java.ClassFile) error {
		if c.IsSynthetic() {
			if debug {
				log.Println("skipping synthetic class:", f.Name, c.Name())
			}
			return nil
		}
		name := convertClassName(c.Name())
		if debug {
			log.Println("Visiting class:", f.Name, name)
		}
		spec.Classes = append(spec.Classes, name)
		return nil
	})
}

func convertClassName(name string) string {
	name = strings.Replace(name, "/", ".", -1)
	name = strings.Replace(name, "$", ".", 1)
	return name
}
