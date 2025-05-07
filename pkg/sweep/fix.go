package sweep

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/stackb/scala-gazelle/pkg/procutil"
)

type DepFixer struct {
	repoDir string
	pkg     string
}

func NewDepFixer(c *config.Config, pkg string) *DepFixer {
	return &DepFixer{
		repoDir: c.RepoRoot,
		pkg:     pkg,
	}
}

func (d *DepFixer) Run() error {
	files, err := d.listChangedScalaFiles()
	if err != nil {
		return err
	}

	log.Println("changed files:", files)

	return nil
}

func (d *DepFixer) listChangedScalaFiles() ([]string, error) {
	args := []string{
		"--work-tree",
		d.repoDir,
		"ls-files",
		"--others",
		"--modified",
		"--full-name",
		"--",
		fmt.Sprintf("%s/*.scala", d.pkg),
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(cmd, err)

	if exitCode != 0 {
		return nil, fmt.Errorf("git ls-files failed: %s", string(output))
	}

	return scanLsFilesOutput(output)
}

func scanLsFilesOutput(output []byte) (files []string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		files = append(files, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return
}
