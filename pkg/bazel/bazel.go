package bazel

import (
	"os"
	"path"
	"regexp"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

// the name of an environment variable at runtime
const TEST_TMPDIR = "TEST_TMPDIR"

var (
	FindBinary   = bazel.FindBinary
	ListRunfiles = bazel.ListRunfiles
)

var nonWordRe = regexp.MustCompile(`\W+`)

func CleanupLabel(in string) string {
	return nonWordRe.ReplaceAllString(in, "_")
}

// NewTmpDir creates a new temporary directory in TestTmpDir().
func NewTmpDir(prefix string) (string, error) {
	if tmp, ok := os.LookupEnv(TEST_TMPDIR); ok {
		err := os.MkdirAll(path.Join(tmp, prefix), 0700)
		return tmp, err
	}
	return os.MkdirTemp("", prefix)
}
