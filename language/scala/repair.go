package scala

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/scala-gazelle/pkg/bazel"
	"github.com/stackb/scala-gazelle/pkg/procutil"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/sweep"
)

type repairMode int

const (
	RepairNone repairMode = iota
	RepairBatch
	RepairWatch
	RepairTransitive
)

var (
	CI = procutil.EnvVar("CI")
)

// String partially implements the flag.Value interface.
func (i *repairMode) String() string {
	switch *i {
	case RepairNone:
		return "none"
	case RepairBatch:
		return "batch"
	case RepairWatch:
		return "watch"
	case RepairTransitive:
		return "transitive"
	}
	return "unknown"
}

func maybeGitCommitAndPushBuildFiles() error {
	if !isCI() {
		return nil
	}
	return gitCommitAndPushBuildFiles("build(gazelle): sweep transitive deps")
}

// Set implements the flag.Value interface.
func (i *repairMode) Set(value string) error {
	switch value {
	case "", "none":
		*i = RepairNone
	case "batch":
		*i = RepairBatch
	case "watch":
		*i = RepairWatch
	case "transitive":
		*i = RepairTransitive
	default:
		return fmt.Errorf("unknown repair value: %s", value)
	}
	return nil
}

func (sl *scalaLang) repair() {
	if err := sl.repairDeps(sl.repairMode); err != nil {
		log.Printf("warning: repair failed: %v", err)
	}
}

func (sl *scalaLang) repairDeps(mode repairMode) error {
	if err := maybeGitCommitAndPushBuildFiles(); err != nil {
		return err
	}

	switch mode {
	case RepairBatch:
		return sl.repairBatch()
	case RepairWatch:
		return sl.repairWatch()
	case RepairTransitive:
		return sl.repairTransitive()
	default:
		return nil
	}
}

func (sl *scalaLang) repairBatch() error {
	rules := gatherResolvableScalaRuleMap(sl.knownRules)
	imports := makeResolvedImports(sl.globalScope)

	fixer := sweep.NewDepFixer(sl.progress, sl.repoRoot, "", rules, imports.Imports, sl, sl.globalScope)
	return fixer.Batch()
}

func (sl *scalaLang) repairWatch() error {
	dir, ok := procutil.LookupEnv(SCALA_GAZELLE_WATCH_DIR)
	if !ok {
		return fmt.Errorf("error: %v must be set to the directory to watch", SCALA_GAZELLE_WATCH_DIR)
	}
	if !path.IsAbs(dir) {
		dir = path.Join(sl.repoRoot, dir)
	}

	rules := gatherResolvableScalaRuleMap(sl.knownRules)
	imports := makeResolvedImports(sl.globalScope)

	fixer := sweep.NewDepFixer(sl.progress, sl.repoRoot, "", rules, imports.Imports, sl, sl.globalScope)

	return fixer.Watch(dir)
}

func (sl *scalaLang) repairTransitive() error {
	rules := gatherResolvableScalaRuleMap(sl.knownRules)
	imports := makeResolvedImports(sl.globalScope)

	fixer := sweep.NewDepFixer(sl.progress, sl.repoRoot, "", rules, imports.Imports, sl, sl.globalScope)

	return fixer.Transitive()
}

func gatherResolvableScalaRuleMap(knownRules map[label.Label]*rule.Rule) sweep.ResolvableScalaRuleMap {
	scalaRules := make(sweep.ResolvableScalaRuleMap)

	for _, knownRule := range knownRules {
		scalaRule, ok := scalarule.GetRule(knownRule)
		if !ok {
			continue
		}
		resolveFunc := knownRule.PrivateAttr("_scala_resolve_closure").(func())
		scalaRules[scalaRule] = resolveFunc
	}

	return scalaRules
}

func isCI() bool {
	return procutil.LookupBoolEnv(CI, false)
}

func gitCommit() error {
	cmd := exec.Command("git", "commit", "-am", "build(gazelle): sweep transitive deps")
	cmd.Dir = bazel.GetBuildWorkspaceDirectory()

	output, err := cmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(cmd, err)
	if exitCode != 0 {
		return fmt.Errorf("git commit failed: %v\n%s", err, string(output))
	}

	return nil
}

func gitHasChanges() (bool, error) {
	// Check for uncommitted changes
	cmd := exec.Command("git", "diff", "--quiet", "HEAD")
	cmd.Dir = bazel.GetBuildWorkspaceDirectory()

	err := cmd.Run()
	exitCode := procutil.CmdExitCode(cmd, err)

	if exitCode == 0 {
		// Exit code 0 means no changes (diff is empty)
		return false, nil
	} else if exitCode == 1 {
		// Exit code 1 from git diff --quiet means changes exist
		return true, nil
	} else {
		// Any other exit code indicates an error
		output, _ := exec.Command("git", "diff", "HEAD").CombinedOutput()
		return false, fmt.Errorf("git diff failed with exit code %d: %v\n%s", exitCode, err, string(output))
	}
}

func gitPush() error {
	cmd := exec.Command("git", "push", "origin", "HEAD")
	cmd.Dir = bazel.GetBuildWorkspaceDirectory()

	output, err := cmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(cmd, err)
	if exitCode != 0 {
		return fmt.Errorf("git push failed: %v\n%s", err, string(output))
	}

	return nil
}

// gitCommitAndPushBuildFiles commits only BUILD.bazel files with the given message
// and skips pre-commit hooks
func gitCommitAndPushBuildFiles(message string) error {
	repoDir := bazel.GetBuildWorkspaceDirectory()

	//
	// FIND CHANGED FILES
	//

	// Find all changed BUILD.bazel files in the repository
	findCmd := exec.Command("git", "ls-files", "--modified", "--exclude-standard", "*BUILD.bazel")
	findCmd.Dir = repoDir

	buildFiles, findErr := findCmd.Output()
	findExitCode := procutil.CmdExitCode(findCmd, findErr)
	if findExitCode != 0 {
		return fmt.Errorf("finding BUILD.bazel files failed: %v", findErr)
	}

	// If no BUILD.bazel files found, nothing to commit
	if len(strings.TrimSpace(string(buildFiles))) == 0 {
		return nil
	}

	//
	// ADD FILES
	//

	// Convert the output to a list of files
	fileList := strings.Fields(string(buildFiles))

	// Add all BUILD.bazel files in a single command
	addCmd := exec.Command("git", append([]string{"add"}, fileList...)...)
	addCmd.Dir = repoDir

	addOutput, addErr := addCmd.CombinedOutput()
	addExitCode := procutil.CmdExitCode(addCmd, addErr)
	if addExitCode != 0 {
		return fmt.Errorf("git add failed: %v\n%s", addErr, string(addOutput))
	}

	//
	// COMMIT
	//

	// Then commit the staged changes, skipping pre-commit hooks
	commitCmd := exec.Command("git", "commit", "-m", message, "--no-verify")
	commitCmd.Dir = repoDir

	// Set environment variables to ensure pre-commit hooks are skipped
	env := os.Environ()
	commitCmd.Env = append(env, "SKIP=1", "GIT_SKIP_HOOKS=1")

	commitOutput, commitErr := commitCmd.CombinedOutput()
	commitExitCode := procutil.CmdExitCode(commitCmd, commitErr)
	if commitExitCode != 0 {
		// Exit code 1 with "nothing to commit" message is normal if no changes were staged
		if strings.Contains(string(commitOutput), "nothing to commit") {
			return nil // No changes to commit, not an error
		}
		return fmt.Errorf("git commit failed: %v\n%s", commitErr, string(commitOutput))
	}

	//
	// PUSH
	//

	pushCmd := exec.Command("git", "push", "origin", "HEAD")
	pushCmd.Dir = repoDir

	pushOutput, err := pushCmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(pushCmd, err)
	if exitCode != 0 {
		return fmt.Errorf("git push failed: %v\n%s", err, string(pushOutput))
	}

	return nil
}
