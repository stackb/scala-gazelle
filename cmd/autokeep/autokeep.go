package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/autokeep"
)

// autokeep is a program that consumes a scala-gazelle cache file, runs 'bazel
// build' on the output, parses it, scans for errors related to missing
// dependencies, and add to deps with "keep" commands where needed.

type config struct {
	cacheFilename   string
	importsFilename string
	bazelExe        string
	rules           []string
	keep            bool
	deps            autokeep.DepsMap
}

func main() {
	log.SetPrefix("autokeep: ")
	log.SetFlags(0) // don't print timestamps

	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func parseFlags(args []string) (*config, error) {
	cfg := new(config)

	fs := flag.NewFlagSet("autokeep", flag.ExitOnError) // flag.ContinueOnError
	fs.StringVar(&cfg.cacheFilename, "cache_file", "", "the scala-gazelle cache file to read")
	fs.StringVar(&cfg.importsFilename, "imports_file", "./scala-gazelle-imports.txt", "the scala-gazelle-imports file to read")
	fs.StringVar(&cfg.bazelExe, "bazel_executable", "bazel", "the path to the bazel executable")
	fs.BoolVar(&cfg.keep, "keep", false, "if true, add # keep comments on needed deps")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: autokeep OPTIONS")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	cfg.rules = fs.Args()

	if cfg.cacheFilename == "" && cfg.importsFilename == "" {
		log.Fatal("at least one of -cache_file or -imports_file is required")
	}

	cfg.deps = make(autokeep.DepsMap)
	if cfg.cacheFilename != "" {
		if err := autokeep.MergeDepsFromCacheFile(cfg.deps, cfg.cacheFilename); err != nil {
			return nil, err
		}
	}
	if cfg.importsFilename != "" {
		if err := autokeep.MergeDepsFromImportsFile(cfg.deps, cfg.importsFilename); err != nil {
			return nil, err
		}
	}
	if len(cfg.deps) == 0 {
		log.Fatalf("label deps map is empty (ensure -imports_file or -cache_file are provided)")
	} else {
		for k := range cfg.deps {
			if strings.HasPrefix(k, "omnistac.postswarm.SelectiveSpotSessionUtils") {
				log.Printf(">> %s", k)
			}
		}
		log.Printf("Loaded %d deps", len(cfg.deps))
	}

	return cfg, nil
}

func run(cfg *config) error {
	args := append([]string{"build", "--keep_going"}, cfg.rules...)

	command := exec.Command(cfg.bazelExe, args...)
	command.Dir = getCommandDir(cfg)
	log.Println(command.String())
	output, err := command.CombinedOutput()

	if err == nil {
		log.Println("PASS")
		return nil
	}

	diagnostics, err := autokeep.ScanOutput(output)
	if err != nil {
		return err
	}

	log.Printf("diagnostics: %+v", diagnostics)
	keep := autokeep.MakeDeltaDeps(cfg.deps, diagnostics)
	log.Printf("keep: %+v", keep)

	if err := autokeep.ApplyDeltaDeps(keep, cfg.keep); err != nil {
		return err
	}

	return nil
}

func getCommandDir(_ *config) string {
	if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
		return bwd
	} else {
		return "."
	}
}
