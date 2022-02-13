package main

import (
	"archive/zip"
	"flag"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/index"
	"github.com/stackb/scala-gazelle/pkg/java"
)

const (
	debug = false
)

var (
	isAnonymous = regexp.MustCompile(`^.*\$[0-9]$`)

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
	if outputFile == "" {
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

	sort.Strings(spec.Classes)

	if err := index.WriteJSONFile(outputFile, spec); err != nil {
		return err
	}
	return nil
}

func parseJarFile(filename string, spec *index.JarSpec) error {
	if debug {
		log.Println("Parsing jar file:", filename)
	}
	entry := java.NewJarClassPathEntry(filename)
	return entry.Visit(func(f *zip.File, c *java.ClassFile) error {
		if c.IsSynthetic() {
			if debug {
				log.Println("skipping synthetic class:", f.Name, c.Name())
			}
			return nil
		}
		name := c.Name()
		if debug {
			log.Println("Visiting class:", f.Name, name)
		}
		// exclude Main$ scala classes
		if strings.HasSuffix(name, "$") {
			if debug {
				log.Println("skipping scala singleton class:", f.Name, c.Name())
			}
			return nil
		}
		// exclude shaded classes
		if strings.Contains(name, "/shaded/") {
			if debug {
				log.Println("skipping shaded class:", f.Name, c.Name())
			}
			return nil
		}
		// exclude anonymous classes like 'com/google/protobuf/Int32Value$1'
		if isAnonymous.MatchString(name) {
			if debug {
				log.Println("skipping anonymous class:", f.Name, c.Name())
			}
			return nil
		}

		name = convertClassName(name)
		spec.Classes = append(spec.Classes, name)
		return nil
	})
}

func convertClassName(name string) string {
	name = strings.Replace(name, "/", ".", -1)
	// name = strings.Replace(name, "$", ".", -1) // TODO(pcj): is this correct to do this with the inner classes?
	return name
}
