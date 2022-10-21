package main

import (
	"embed"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var assets embed.FS

type options struct {
	debug                bool
	restoreEmbeddedFiles bool
	nodeBinPath          string
	nodePath             string
	scriptPath           string
	files                []string
}

func parseOptions(args []string) (*options, error) {
	var opts options
	var fs flag.FlagSet
	fs.StringVar(&opts.nodeBinPath, "node_bin_path", "", "override location of node binary")
	fs.StringVar(&opts.nodePath, "node_path", "", "override location of node modules")
	fs.StringVar(&opts.scriptPath, "script_path", "", "override location of sourceindexer script")
	fs.BoolVar(&opts.restoreEmbeddedFiles, "embedded", true, "restore embedded files")
	fs.BoolVar(&opts.debug, "debug", false, "debug mode")
	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}
	opts.files = fs.Args()

	if opts.restoreEmbeddedFiles {
		tmpDir := os.TempDir()
		// mustRestore(&opts, tmpDir, embeddedInterpreter)
		// mustRestore(&opts, tmpDir, embeddedAssets)
		if opts.nodeBinPath == "" {
			opts.nodeBinPath = filepath.Join(tmpDir, "external/nodejs_darwin_amd64/bin/nodejs/bin/node")
		}
		if opts.nodePath == "" {
			opts.nodePath = filepath.Join(tmpDir, "cmd/sourceindexer")
		}
		if opts.scriptPath == "" {
			opts.scriptPath = filepath.Join(tmpDir, "cmd/sourceindexer/sourceindexer.js")
		}
	}

	// in the case where this tool is being run as an action the paths are
	// relative to the runfiles dir.
	runfilesDir := args[0] + ".runfiles"
	if opts.nodeBinPath == "" {
		opts.nodeBinPath = filepath.Join(runfilesDir, "nodejs_darwin_amd64/bin/nodejs/bin/node")
	}
	if opts.nodePath == "" {
		opts.nodePath = filepath.Join(runfilesDir, "scala_gazelle/cmd/sourceindexer")
	}
	if opts.scriptPath == "" {
		opts.scriptPath = filepath.Join(runfilesDir, "scala_gazelle/cmd/sourceindexer/sourceindexer.js")
	}

	return &opts, nil
}

func main() {
	log.SetPrefix("sourceindexer.go: ")
	log.SetFlags(0) // don't print timestamps

	opts, err := parseOptions(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if opts.debug {
		// log.Println(os.Environ())
		listFiles(".")
	}

	args := append([]string{opts.scriptPath}, opts.files...)
	env := []string{"NODE_PATH=" + opts.nodePath}

	exitCode, err := run(opts.nodeBinPath, args, ".", env)
	if err != nil {
		log.Print(err)
	}
	os.Exit(exitCode)
}

// run a command
func run(entrypoint string, args []string, dir string, env []string) (int, error) {
	cmd := exec.Command(entrypoint, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.Dir = dir
	err := cmd.Run()

	var exitCode int
	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			log.Printf("Could not get exit code for failed program: %v, %v", entrypoint, args)
			exitCode = -1
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	return exitCode, err
}

// listFiles - convenience debugging function to log the files under a given dir
func listFiles(dir string) error {
	log.Println("Listing files under " + dir)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		log.Println(path)
		return nil
	})
}

// mustRestore - Restore assets.
func mustRestore(opts *options, baseDir string, assets map[string][]byte) {
	// unpack variable is provided by the go_embed data and is a
	// map[string][]byte such as {"/usr/share/games/fortune/literature.dat":
	// bytes... }
	for rel, bytes := range assets {
		filename := filepath.Join(baseDir, rel)
		dirname := filepath.Dir(filename)
		// log.Printf("file %s, dir %s, rel %d, abs %s, absdir: %s", file, dir, rel, abs, absdir)
		if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
			log.Fatalf("Failed to create asset dir %s: %v", dirname, err)
		}
		if err := ioutil.WriteFile(filename, bytes, os.ModePerm); err != nil {
			log.Fatalf("Failed to write asset %s: %v", filename, err)
		}

		switch filepath.Base(rel) {
		case "sourceindexer.js":
			opts.scriptPath = filename
		case "node":
			opts.nodeBinPath = filename
		case "package.json":
			// cmd/sourceindexer/node_modules/scalameta-parsers/package.json -> cmd/sourceindexer/
			opts.nodePath = filepath.Dir(filepath.Dir(dirname))
		}

		if opts.debug {
			log.Printf("Restored %s", filename)
		}
	}
}
