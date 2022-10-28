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
