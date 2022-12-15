package parser

import (
	"io"
	"log"
	"os/exec"
	"syscall"

	"github.com/amenzhinsky/go-memexec"
)

// ExecJS runs the embedded js runtime interpreter
func ExecJS(dir string, args, env []string, in io.Reader, stdout, stderr io.Writer) (int, error) {
	exe, err := memexec.New(nodeExe)
	if err != nil {
		return 1, err
	}
	defer exe.Close()

	cmd := exe.Command(args...)
	cmd.Stdin = in
	cmd.Stdout = stdout
	cmd.Stderr = stderr
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
			log.Printf("Could not get exit code for failed process, %v", args)
			exitCode = -1
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	return exitCode, err
}
