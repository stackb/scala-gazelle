package bazel

import (
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/stackb/scala-gazelle/pkg/procutil"
)

// the name of an environment variable at runtime
const TEST_TMPDIR = "TEST_TMPDIR"

var (
	FindBinary   = bazel.FindBinary
	ListRunfiles = bazel.ListRunfiles
	nonWordRe    = regexp.MustCompile(`\W+`)
)

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

func ExecCommand(bazelExe, command string, labels ...string) ([]byte, int, error) {
	args := append([]string{command}, labels...)

	cmd := exec.Command(bazelExe, args...)
	cmd.Dir = GetBuildWorkspaceDirectory()

	log.Println("🧱", cmd.String())
	output, err := cmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(cmd, err)

	return output, exitCode, err
}

func GetBuildWorkspaceDirectory() string {
	if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
		return bwd
	} else {
		return "."
	}
}
