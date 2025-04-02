package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

var outputFile string

func main() {
	log.SetPrefix("scalafilemerge: ")
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
	fs := flag.NewFlagSet("scalafilemerge", flag.ExitOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: scalafilemerge @PARAMS_FILE | scalafilemerge OPTIONS FILES")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return nil, err
	}

	if outputFile == "" {
		log.Fatal("-output_file is required")
	}

	files = fs.Args()
	// if len(files) == 0 {
	// 	err = fmt.Errorf("scalafilemerge positional args should be a non-empty list of files to merge")
	// }

	return
}

func merge(filenames ...string) (*sppb.RuleSet, error) {
	ruleSet := new(sppb.RuleSet)

	for _, filename := range filenames {
		var rule sppb.Rule
		err := protobuf.ReadFile(filename, &rule)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %v", filename, err)
		}
		ruleSet.Rules = append(ruleSet.Rules, &rule)
	}

	sort.Slice(ruleSet.Rules, func(i, j int) bool {
		a := ruleSet.Rules[i].Label
		b := ruleSet.Rules[j].Label
		return a < b
	})

	return ruleSet, nil
}
