package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/protobuf"

	wppb "github.com/stackb/scala-gazelle/blaze/worker"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

type Config struct {
	OutputFile       string
	RuleLabel        string
	RuleKind         string
	PersistentWorker bool
	SourceFiles      []string
	Parser           *parser.ScalametaParser
	Cwd              string
}

func main() {
	log.SetPrefix("scalafileextract: ")
	log.SetOutput(os.Stderr)
	log.SetFlags(0) // don't print timestamps

	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	log.Println("args:", args)

	parsedArgs, err := collections.ReadArgsParamsFile(args)
	log.Println("parsedArgs:", parsedArgs)
	if err != nil {
		return fmt.Errorf("failed to read params file: %v", err)
	}

	cfg, err := parseFlags(parsedArgs)
	if err != nil {
		return fmt.Errorf("failed to parse args: %v", err)
	}

	cfg.Parser = parser.NewScalametaParser(
		parser.WithHttpClientTimeout(60 * time.Second),
	)

	if err := cfg.Parser.Start(); err != nil {
		return fmt.Errorf("starting parser: %w", err)
	}
	defer func() {
		cfg.Parser.Stop()
	}()

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting os cwd: %v", err)
	} else {
		cfg.Cwd = wd
	}

	if cfg.PersistentWorker {
		if err := persistentWork(&cfg); err != nil {
			return fmt.Errorf("while performing persistent work: %v", err)
		}
	} else {
		if err := batchWork(&cfg); err != nil {
			return fmt.Errorf("while performing batch work: %v", err)
		}
	}

	return nil
}

func persistentWork(cfg *Config) error {
	for {
		var req wppb.WorkRequest
		if err := protobuf.ReadDelimitedFrom(&req, os.Stdin); err != nil {
			if err == io.EOF {
				// this is the signal to terminate the program
				break
			}
			return fmt.Errorf("reading work request: %v", err)
		}

		var resp wppb.WorkResponse
		resp.RequestId = req.RequestId

		batchCfg, err := parseFlags(req.Arguments)
		if err != nil {
			return fmt.Errorf("parsing work request arguments: %v", err)
		}

		batchCfg.Parser = cfg.Parser
		batchCfg.Cwd = cfg.Cwd

		if err := batchWork(&batchCfg); err != nil {
			return fmt.Errorf("performing persistent batch!: %v", err)
		}

		if err := protobuf.WriteDelimitedTo(&resp, os.Stdout); err != nil {
			return fmt.Errorf("writing work response: %v", err)
		}
	}
	return nil
}

func batchWork(cfg *Config) error {
	if cfg.OutputFile == "" {
		return fmt.Errorf("--output_file is required")
	}
	if cfg.RuleLabel == "" {
		return fmt.Errorf("--rule_label is required")
	}
	if cfg.RuleKind == "" {
		return fmt.Errorf("--rule_kind is required")
	}
	if len(cfg.SourceFiles) == 0 {
		return fmt.Errorf("source files list must not be empty")
	}

	now := time.Now()
	fail := func(err error) error {
		return fmt.Errorf("%v (%v)", err, time.Since(now))
	}

	files, err := extract(cfg.Parser, cfg.Cwd, cfg.SourceFiles)
	if err != nil {
		return fail(fmt.Errorf("failed to extract files: %v", err))
	}

	rule := sppb.Rule{
		Label: cfg.RuleLabel,
		Kind:  cfg.RuleKind,
		Files: files,
	}

	if err := protobuf.WriteFile(cfg.OutputFile, &rule); err != nil {
		return fail(fmt.Errorf("failed to write output file: %v", err))
	}

	return nil
}

func parseFlags(args []string) (cfg Config, err error) {
	fs := flag.NewFlagSet("scalafileextract", flag.ExitOnError)
	fs.StringVar(&cfg.OutputFile, "output_file", "", "the output file to write")
	fs.StringVar(&cfg.RuleLabel, "rule_label", "", "the rule label being parsed")
	fs.StringVar(&cfg.RuleKind, "rule_kind", "", "the rule kind being parsed")
	fs.BoolVar(&cfg.PersistentWorker, "persistent_worker", false, "present if this tool is being invokes as a bazel persistent worker")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: scalafileextract @PARAMS_FILE")
		fs.PrintDefaults()
	}

	if err = fs.Parse(args); err != nil {
		return
	}
	cfg.SourceFiles = fs.Args()
	return
}

func extract(parser *parser.ScalametaParser, dir string, sourceFiles []string) ([]*sppb.File, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := parser.Parse(ctx, request)
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
