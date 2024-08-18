package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"

	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

var (
	outputFile string
	infoFiles  collections.StringSlice
)

type infoFile struct {
	source string
	path   string
}

func main() {
	log.SetPrefix("semanticdbidx: ")
	log.SetFlags(0) // don't print timestamps

	args, err := collections.ReadOsArgsParams()
	if err != nil {
		log.Fatalln("failed to read params file:", err)
	}

	roots, err := parseFlags(args)
	if err != nil {
		log.Fatal(err)
	}

	index, err := index(roots...)
	if err != nil {
		log.Fatal(err)
	}

	if err := protobuf.WriteFile(outputFile, index); err != nil {
		log.Fatal(err)
	}
}

func parseFlags(args []string) (files []infoFile, err error) {
	fs := flag.NewFlagSet("semanticdbidx", flag.ExitOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Var(&infoFiles, "info_file", "a string NAME=RELPATH that maps NAME (the name of the scala source file) to RELPATH (the relative path of the semanticdb info file for that source file)")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: semanticdbidx @PARAMS_FILE | semanticdbidx OPTIONS FILES")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return
	}

	if outputFile == "" {
		return nil, fmt.Errorf("-output_file is required")
	}
	if len(infoFiles) == 0 {
		return nil, fmt.Errorf("at least one -info_file is required")
	}

	for _, arg := range infoFiles {
		parts := strings.SplitN(arg, "=", 2)
		info := infoFile{source: parts[0], path: parts[1]}
		files = append(files, info)
	}

	return
}

func index(files ...infoFile) (*spb.InfoMap, error) {
	index := new(spb.InfoMap)
	index.Entries = make(map[string]string)

	for _, file := range files {
		index.Entries[file.source] = file.path
	}

	return index, nil
}
