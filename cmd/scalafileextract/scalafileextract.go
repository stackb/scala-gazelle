package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/protobuf"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

var (
	outputFile string
	ruleLabel  string
	ruleKind   string
)

type parseContext struct {
	parser *parser.ScalametaParser
}

func main() {
	log.SetPrefix("scalafileextract: ")
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

	sourceFiles, err := parseFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %v", err)
	}

	parser := parser.NewScalametaParser()
	if err := parser.Start(); err != nil {
		return fmt.Errorf("starting parser: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	files, err := extract(&parseContext{parser}, cwd, sourceFiles)
	if err != nil {
		return fmt.Errorf("failed to extract files: %v", err)
	}

	rule := sppb.Rule{
		Label: ruleLabel,
		Kind:  ruleKind,
		Files: files,
	}

	if err := protobuf.WriteFile(outputFile, &rule); err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}
	// if err := protobuf.WriteStableJSONFile(outputFile, &rule); err != nil {
	// 	return fmt.Errorf("failed to write output file: %v", err)
	// }

	return nil
}

func parseFlags(args []string) (files []string, err error) {
	fs := flag.NewFlagSet("scalafileextract", flag.ExitOnError)
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.StringVar(&ruleLabel, "rule_label", "", "the rule label being parsed")
	fs.StringVar(&ruleKind, "rule_kind", "", "the rule kind being parsed")

	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: scalafileextract @PARAMS_FILE | scalafileextract OPTIONS")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return nil, err
	}

	if outputFile == "" {
		log.Fatal("-output_file is required")
	}
	if ruleLabel == "" {
		log.Fatal("-rule_label is required")
	}
	if ruleKind == "" {
		log.Fatal("-rule_kind is required")
	}

	files = fs.Args()
	if len(files) == 0 {
		err = fmt.Errorf("positional args should not be empty")
	}

	return
}

func extract(ctx *parseContext, dir string, sourceFiles []string) ([]*sppb.File, error) {
	request := &sppb.ParseRequest{
		Filenames: make([]string, len(sourceFiles)),
	}

	//
	// the parser cwd is in a temp dir and needs absolute paths.  Use a map to
	// reset the paths to the relative form before returning.
	//
	filenames := make(map[string]string)
	for i, sourceFile := range sourceFiles {
		filename := path.Join(dir, sourceFile)
		request.Filenames[i] = filename
		filenames[filename] = sourceFile
	}

	response, err := ctx.parser.Parse(context.Background(), request)
	if err != nil {
		return nil, err
	}

	for _, file := range response.Files {
		if file.Error != "" {
			return nil, fmt.Errorf(file.Error)
		}
		file.Filename = filenames[file.Filename]
	}

	return response.Files, nil
}
