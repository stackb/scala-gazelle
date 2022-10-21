package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/amenzhinsky/go-memexec"
)

func main() {
	log.SetPrefix("sourceindexer.go: ")
	log.SetFlags(0) // don't print timestamps

	tmpDir := os.TempDir()

	opts, err := ParseOptions(tmpDir, os.Args[0], os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if opts.Debug {
		// log.Println(os.Environ())
		listFiles(".")
	}

	args := append([]string{opts.ScriptPath}, opts.Files...)
	env := []string{"NODE_PATH=" + opts.NodePath}

	exitCode, err := run(opts.NodeBinPath, args, ".", env)
	if err != nil {
		log.Print(err)
	}
	os.Exit(exitCode)
}

// run a command
func run(entrypoint string, args []string, dir string, env []string) (int, error) {
	exe, err := memexec.New(nodeExe)
	if err != nil {
		return 1, err
	}
	defer exe.Close()

	cmd := exe.Command(args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.Dir = dir
	err = cmd.Run()

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
