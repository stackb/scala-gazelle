package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/stackb/scala-gazelle/pkg/wildcardimport"
)

const (
	executableName = "wildcardimportfixer"
)

type config struct {
	ruleLabel      string
	targetFilename string
	importPrefix   string
	bazelExe       string
}

func main() {
	log.SetPrefix(executableName + ": ")
	log.SetFlags(0) // don't print timestamps

	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if err := run(cfg); err != nil {
		log.Fatalln("ERROR:", err)
	}

}

func parseFlags(args []string) (*config, error) {
	cfg := new(config)

	fs := flag.NewFlagSet(executableName, flag.ExitOnError)
	fs.StringVar(&cfg.ruleLabel, "rule_label", "", "the rule label to iteratively build")
	fs.StringVar(&cfg.targetFilename, "target_filename", "", "the scala file to fix")
	fs.StringVar(&cfg.importPrefix, "import_prefix", "", "the scala import prefix to set")
	fs.StringVar(&cfg.bazelExe, "bazel_executable", "bazel", "the path to the bazel executable")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s OPTIONS", executableName)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.ruleLabel == "" {
		log.Fatal("-rule_label is required")
	}

	return cfg, nil
}

func run(cfg *config) error {

	var err error

	fixer := wildcardimport.NewFixer(&wildcardimport.FixerOptions{
		BazelExecutable: cfg.bazelExe,
	})

	symbols, err := fixer.Fix(&wildcardimport.FixConfig{
		RuleLabel:    cfg.ruleLabel,
		Filename:     cfg.targetFilename,
		ImportPrefix: cfg.importPrefix,
	})
	if err != nil {
		return err
	}

	log.Println("FIXED", cfg.targetFilename, symbols)

	return nil
}
