package wildcardimport

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func execBazelBuild(bazelExe string, label string) ([]byte, int, error) {
	args := []string{"build", label}

	command := exec.Command(bazelExe, args...)
	command.Dir = GetBuildWorkspaceDirectory()

	log.Println("🧱", command.String())
	output, err := command.CombinedOutput()
	if err != nil {
		// log.Println("cmdErr:", err)
		// Check for exit errors specifically
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode := waitStatus.ExitStatus()
			return output, exitCode, err
		} else {
			return output, -1, err
		}
	}
	return output, 0, nil
}

func GetBuildWorkspaceDirectory() string {
	if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
		return bwd
	} else {
		return "."
	}
}
