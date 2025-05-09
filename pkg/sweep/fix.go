package sweep

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/pcj/mobyprogress"
	"github.com/stackb/scala-gazelle/pkg/autokeep"
	"github.com/stackb/scala-gazelle/pkg/bazel"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/procutil"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalarule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

type ResolvableScalaRuleMap map[scalarule.Rule]func()

type DepFixer struct {
	// progress is the progress interface
	progress mobyprogress.Output
	// repoRoot is the absolute path to the repository root
	repoRoot string
	// pkg is the relative path from repoDir
	pkg string
	// resolvers is a map that contains the resolver function per scala rule
	resolvers ResolvableScalaRuleMap
	// files is a map that helps identify the rule to which a file belongs
	rules map[*sppb.File]scalarule.Rule
	// files is a map that helps identify the file by filename
	files map[string]*sppb.File
	// scope is the (global) scope that should be used to resolve symbols
	imports autokeep.DepsMap
	// global knownScopes
	knownScopes resolver.KnownScopeRegistry
	// scala file Parser

}

func NewDepFixer(progress mobyprogress.Output, repoRoot, pkg string, resolvers ResolvableScalaRuleMap, imports map[string]string, knownScopes resolver.KnownScopeRegistry) *DepFixer {
	fixer := &DepFixer{
		progress:    progress,
		repoRoot:    repoRoot,
		pkg:         pkg,
		resolvers:   resolvers,
		rules:       make(map[*sppb.File]scalarule.Rule),
		files:       make(map[string]*sppb.File),
		imports:     imports,
		knownScopes: knownScopes,
	}
	for r := range resolvers {
		files := r.Rule().Files
		for _, f := range files {
			fixer.rules[f] = r
			fixer.files[f.Filename] = f
		}
	}
	return fixer
}

func (d *DepFixer) Batch() error {
	changedFiles, err := listChangedScalaFiles(d.repoRoot, d.pkg)
	if err != nil {
		return err
	}

	log.Println("changed files:", changedFiles)

	// toBuild helps gather the list of rules that need to be built based on the
	// list of changed files
	toBuild := make(map[string]scalarule.Rule)

	for _, filename := range changedFiles {
		if file, ok := d.files[filename]; ok {
			rule := d.rules[file]
			toBuild[rule.Rule().Label] = rule
		} else {
			log.Println("no scala build rule known for:", file)
		}
	}

	for label, rule := range toBuild {
		// re-parse the srcs
		if err := rule.ParseSrcs(); err != nil {
			return fmt.Errorf("parse %s failed: %v", label, err)
		}
		// call the resolver for the rule
		d.resolvers[rule]()
	}
	if err := d.fix(toBuild); err != nil {
		return fmt.Errorf("building rule: %v", err)
	}

	return nil
}

func (d *DepFixer) fix(toBuild map[string]scalarule.Rule) error {
	labels := make([]string, 0, len(toBuild))
	for label := range toBuild {
		labels = append(labels, label)
	}
	if len(labels) == 0 {
		return nil
	}
	sort.Strings(labels)

	log.Println("fixing:", labels)

	writeBuildProgress(d.progress, fmt.Sprintf("> bazel build %s", strings.Join(labels, " ")))

	out, exitCode, _ := bazel.ExecCommand("bazel", "build", labels...)
	if exitCode == 0 {
		log.Println("builds cleanly (no action needed)")
		return nil
	}

	diagnostics, err := autokeep.ScanOutput(out)
	if err != nil {
		return fmt.Errorf("failed to scan build output: %v", err)
	}

	delta := autokeep.MakeDeltaDeps(diagnostics, d.imports, d.files, d.knownScopes)

	toAdd := make(map[string][]string)
	toRemove := make(map[string][]string)

	for _, ruleDeps := range delta.Add {
		toBuild[ruleDeps.Label] = nil
		toAdd[ruleDeps.Label] = append(toAdd[ruleDeps.Label], ruleDeps.Deps...)
	}
	for _, ruleDeps := range delta.Remove {
		toBuild[ruleDeps.Label] = nil
		toRemove[ruleDeps.Label] = append(toRemove[ruleDeps.Label], ruleDeps.Deps...)
	}

	// if no actions could be derived, the build failed and we don't know what to do.
	if len(toAdd) == 0 && len(toRemove) == 0 {
		return fmt.Errorf("build failed, but the corrective action(s) could not be determined.  Manual intervention is required:\n%s", string(out))
	}

	for ruleLabel, deps := range toAdd {
		deps := collections.SliceDeduplicate(deps)
		log.Printf("buildozer 'add deps %s' %s", strings.Join(deps, " "), ruleLabel)
		if err := runBuildozer(
			d.progress,
			fmt.Sprintf("add deps %s", strings.Join(deps, " ")),
			ruleLabel,
		); err != nil {
			return err
		}
	}
	for ruleLabel, deps := range toRemove {
		deps := collections.SliceDeduplicate(deps)
		log.Printf("buildozer 'remove deps %s' %s", strings.Join(deps, " "), ruleLabel)
		if err := runBuildozer(
			d.progress,
			fmt.Sprintf("remove deps %s", strings.Join(deps, " ")),
			ruleLabel,
		); err != nil {
			return err
		}
	}

	return d.fix(toBuild)
}

func runBuildozer(progress mobyprogress.Output, args ...string) error {
	writeBuildProgress(progress, fmt.Sprintf("> buildozer %s", strings.Join(args, " ")))

	cmd := exec.Command("buildozer", args...)
	cmd.Dir = bazel.GetBuildWorkspaceDirectory()

	output, err := cmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(cmd, err)

	if exitCode != 0 {
		return fmt.Errorf("buildozer failed with exit code %d: %s", exitCode, string(output))
	}

	return nil
}

func listChangedScalaFiles(repoDir, pkg string) ([]string, error) {
	args := []string{
		"--work-tree",
		repoDir,
		"ls-files",
		"--others",
		"--modified",
		"--full-name",
		"--",
		path.Join(pkg, "*.scala"),
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	exitCode := procutil.CmdExitCode(cmd, err)

	if exitCode != 0 {
		return nil, fmt.Errorf("%s failed: %s", args, string(output))
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

func writeBuildProgress(output mobyprogress.Output, message string) {
	output.WriteProgress(mobyprogress.Progress{
		ID:      "build",
		Message: message,
	})
}
