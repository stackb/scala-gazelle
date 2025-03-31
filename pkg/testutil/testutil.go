package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/stackb/scala-gazelle/pkg/procutil"
)

func MustReadAndPrepareTestFiles(t *testing.T, files []testtools.FileSpec) (tmpDir string, filenames []string, clean func()) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := range files {
		if files[i].Content == "" {
			files[i].Content = MustReadTestFile(t, cwd, files[i].Path)
		}
	}
	return MustPrepareTestFiles(t, files)
}

func MustPrepareTestFiles(t *testing.T, files []testtools.FileSpec) (tmpDir string, filenames []string, clean func()) {
	tmpDir = t.TempDir()

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	filenames = MustWriteTestFiles(t, tmpDir, files)

	return tmpDir, filenames, cleanup
}

func MustWriteTestFiles(t *testing.T, tmpDir string, files []testtools.FileSpec) []string {
	var filenames []string
	for _, file := range files {
		abs := filepath.Join(tmpDir, file.Path)
		dir := filepath.Dir(abs)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		if !file.NotExist {
			if err := os.WriteFile(abs, []byte(file.Content), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}
		filenames = append(filenames, abs)
	}
	return filenames
}

func MustReadTestFile(t *testing.T, dir string, filename string) string {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		ListFiles(t, dir)
		t.Fatal("reading", filename, ":", err)
	}
	return string(data)
}

// EqualError reports whether errors a and b are considered equal.
// They're equal if both are nil, or both are not nil and a.Error() == b.Error().
func EqualError(a, b error) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Error() == b.Error()
}

// ExpectError asserts that the errors are equal.  Return value is true
// if the "want" argument is non-nil.
func ExpectError(t *testing.T, want, got error) bool {
	if !EqualError(want, got) {
		t.Fatal("errors: want:", want, "got:", got)
	}
	return want != nil
}

// ListFiles is a convenience debugging function to log the files under a given dir.
func ListFiles(t *testing.T, dir string) {
	t.Log("Listing files under:", dir)
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		t.Log(path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

type GazelleResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunGazelle executes the gazelle command with the specified working directory,
// environment variables, and command-line arguments. It returns the command
// output (stdout and stderr) and any error that occurred.
func RunGazelle(t *testing.T, workingDir string, env []string, args ...string) (*GazelleResult, error) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	gazelle := filepath.Join(cwd, "gazelle")

	// Find gazelle in PATH or use an absolute path if needed
	gazelleCmd, err := exec.LookPath(gazelle)
	if err != nil {
		return nil, fmt.Errorf("gazelle command not found in PATH: %w", err)
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
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()
	if err != nil {
		t.Logf("command error: %v", err)
	}

	exitCode := procutil.CmdExitCode(cmd, err)

	return &GazelleResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, err
}
