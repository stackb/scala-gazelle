package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// cycler is a migration tool.  It executes a loop that spawns an external
// script until it succeeds.  After each failure, the build events are parsed
// and buildozer commands are appended to a file.  Is it assumed that the
// external script will apply the buildozer commands and re-run bazel with a
// desired list of targets.  Ideally the program iterates until all missing
// types are recovered and all targets build successfully.  The program requires
// a csv file that maps import symbols to the providing label.

func main() {
	log.SetPrefix("cycler: ")
	log.SetFlags(0) // don't print timestamps

	fs := flag.NewFlagSet("cycler", flag.ContinueOnError)

	cfg := newConfig(fs)

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	if err := cfg.validate(); err != nil {
		log.Fatal(err)
	}

	if err := cycle(cfg); err != nil {
		log.Fatal(err)
	}

	log.Println("SUCCESS!")
}

func cycle(cfg *config) error {
	n := 0
	for {
		if cfg.iterations != 0 && n == cfg.iterations {
			return fmt.Errorf("iteration limit reached: %d", cfg.iterations)
		}
		if err := iterate(cfg, n); err != nil {
			return err
		}
		n += 1
	}
	return nil
}

func iterate(cfg *config, n int) error {
	log.Printf("--- ITERATION %d ---", n+1)

	if err := runCommand(cfg.shellFile, cfg.scriptFile, strings.Fields(cfg.scriptArgs)); err == nil {
		return nil
	}

	env := newEnv(cfg)

	// errs is a list of errors, but not necessarily fatal ones
	commands, errs := env.run()
	if len(errs) > 0 {
		log.Printf("Got %d errors during iteration #%d", len(errs), n+1)
		for _, err := range errs {
			log.Print(err)
		}
	}

	if len(commands) == 0 {
		return fmt.Errorf("Failed to make progress.  Aborting")
	}

	if err := cfg.writeBuildozerCommandFile(commands); err != nil {
		return err
	}

	return nil
}

// runCommand executes a bash subprocess that inherits stdout, stderr, and the
// environment from this process.
func runCommand(shell, script string, args []string) error {
	args = append([]string{script}, args...)
	cmd := exec.Command(shell, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println(">", shell, args)

	return cmd.Run()
}
