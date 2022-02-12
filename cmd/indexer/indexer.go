package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/java"
)

// Consider:
// - https://github.com/akrylysov/pogreb
// - https://github.com/openacid/slim

const (
	debug = false
)

type config struct {
	inputFile  string
	outputFile string
}

type Input struct {
	Label string   `json:"label,omitempty"`
	Jars  []string `json:"jars,omitempty"`
}

type Output struct {
	Label   string   `json:"label,omitempty"`
	Classes []string `json:"classes,omitempty"`
}

func main() {
	log.SetPrefix("jarindexer: ")
	log.SetFlags(0) // don't print timestamps

	conf := config{}
	fs := flag.NewFlagSet("jarindexer", flag.ContinueOnError)

	fs.StringVar(&conf.inputFile, "input_file", "", "the input configuration file")
	fs.StringVar(&conf.outputFile, "output_file", "", "the output file to write")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	if debug {
		log.Printf("Reading config file %q", conf.inputFile)
		log.Printf("Writing output file %q", conf.outputFile)
	}

	if err := run(&conf); err != nil {
		log.Fatal(err)
	}
}

func run(conf *config) error {
	listFiles(".")
	in, err := readInputFile(conf.inputFile)
	if err != nil {
		return err
	}
	out := &Output{
		Label: in.Label,
	}
	for _, jar := range in.Jars {
		if err := parseJarFile(jar, out); err != nil {
			log.Printf("warning: could not parse %s: %v", jar, err)
			// return fmt.Errorf("parsing jar %s: %w", jar, err)
		}
	}
	if err := writeOutputFile(conf.outputFile, out); err != nil {
		return err
	}
	return nil
}

func readInputFile(filename string) (*Input, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var in Input
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &in, nil
}

func writeOutputFile(filename string, out *Output) error {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0o644)
}

func parseJarFile(filename string, out *Output) error {
	log.Println("Parsing jar file:", filename)
	entry := java.NewJarClassPathEntry(filename)
	return entry.Visit(func(f *zip.File, c *java.ClassFile) error {
		if c.IsSynthetic() {
			log.Println("skipping synthetic class:", f.Name, c.Name())
			return nil
		}
		name := convertClassName(c.Name())
		log.Println("Visiting class:", f.Name, name)
		out.Classes = append(out.Classes, name)
		return nil
	})
}

func convertClassName(name string) string {
	name = strings.Replace(name, "/", ".", -1)
	name = strings.Replace(name, "$", ".", 1)
	return name
}

// listFiles - convenience debugging function to log the files under a given dir
func listFiles(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if info.Mode()&os.ModeSymlink > 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			log.Printf("%s -> %s", path, link)
			return nil
		}

		log.Println(path)
		return nil
	})
}
