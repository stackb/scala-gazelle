package files

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// RunGazelle executes the gazelle command with the specified working directory,
// environment variables, and command-line arguments. It returns the command
// output (stdout and stderr) and any error that occurred.
func RunGazelle(t *testing.T, workingDir string, env []string, args ...string) (string, error) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	gazelle := filepath.Join(cwd, "gazelle")

	// Find gazelle in PATH or use an absolute path if needed
	gazelleCmd, err := exec.LookPath(gazelle)
	if err != nil {
		return "", fmt.Errorf("gazelle command not found in PATH: %w", err)
	}

	// Create the command with the provided arguments
	cmd := exec.Command(gazelleCmd, args...)

	// Set working directory
	cmd.Dir = workingDir

	// Set environment variables (appending to or replacing the current environment)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}

	// Capture both stdout and stderr
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Run the command
	err = cmd.Run()

	t.Log("output:", output.String())
	return output.String(), err
}
