package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func MustPrepareTestFiles(t *testing.T, files []testtools.FileSpec) (tmpDir string, filenames []string, clean func()) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		t.Fatal(err)
	}

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
			if err := ioutil.WriteFile(abs, []byte(file.Content), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}
		filenames = append(filenames, abs)
	}
	return filenames
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
