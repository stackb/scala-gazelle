package wildcardimport

import (
	"log"
	"os"
	"os/exec"

	"github.com/stackb/scala-gazelle/pkg/procutil"
)

func execBazelBuild(bazelExe string, label string) ([]byte, int, error) {
	args := []string{"build", label}

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
