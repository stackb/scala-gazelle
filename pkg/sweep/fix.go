package sweep

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"

	"github.com/fsnotify/fsnotify"
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
	// the global scope
	globalScope resolver.Scope
}

func NewDepFixer(progress mobyprogress.Output, repoRoot, pkg string, resolvers ResolvableScalaRuleMap, imports map[string]string, knownScopes resolver.KnownScopeRegistry, globalScope resolver.Scope) *DepFixer {
	fixer := &DepFixer{
		progress:    progress,
		repoRoot:    repoRoot,
		pkg:         pkg,
		resolvers:   resolvers,
		rules:       make(map[*sppb.File]scalarule.Rule),
		files:       make(map[string]*sppb.File),
		imports:     imports,
		knownScopes: knownScopes,
		globalScope: globalScope,
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

func (d *DepFixer) Watch(dir string) error {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) {
					rel := strings.TrimPrefix(strings.TrimPrefix(event.Name, d.repoRoot), "/")
					log.Println("modified file:", event.Name, rel)
					if err := d.handleChangedFiles(rel); err != nil {
						log.Println("watch error:", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("watching", dir)

	// Block main goroutine forever.
	<-make(chan struct{})

	return nil
}

func (d *DepFixer) Batch() error {
	changedFiles, err := listChangedScalaFiles(d.repoRoot, d.pkg)
	if err != nil {
		return err
	}

	return d.handleChangedFiles(changedFiles...)
}

func (d *DepFixer) handleChangedFiles(changedFiles ...string) error {

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

	targets := make([]string, 0, len(toBuild))
	for label, rule := range toBuild {
		targets = append(targets, label)
		// re-parse the srcs
		if err := rule.ParseSrcs(); err != nil {
			return fmt.Errorf("parse %s failed: %v", label, err)
		}
		// call the resolver for the rule
		d.resolvers[rule]()
	}
	// if err := d.Repair(targets...); err != nil {
	// 	return fmt.Errorf("building rule: %v", err)
	// }

	return nil
}

type RepairHandler interface {
	// Targets returns the list of build targets that should be repaired
	Targets() []string
	// Add is a callback when the given target should add (missing) deps
	Add(target string, deps []string)
	// Remove is a callback when the given target should remove (unused) deps
	Remove(target string, deps []string)
	// Apply signals that the current actions should be applied.
	Apply(iteration int) error
	// Done signals that the repair process has completed
	Done()
}

func (d *DepFixer) Repair(handler RepairHandler) error {
	return d.repairIteration(handler, 1)
}

func (d *DepFixer) repairIteration(handler RepairHandler, iteration int) error {
	targets := collections.DeduplicateAndSort(handler.Targets())
	if len(targets) == 0 {
		return nil
	}

	log.Println("fixing:", targets)

	writeBuildProgress(d.progress, fmt.Sprintf("> bazel build %s", strings.Join(targets, " ")))

	out, exitCode, _ := bazel.ExecCommand("bazel", "build", targets...)
	if exitCode == 0 {
		log.Println("builds cleanly (no action needed)")
		handler.Done()
		return nil
	}

	diagnostics, err := autokeep.ScanOutput(out)
	if err != nil {
		return fmt.Errorf("failed to scan build output: %v", err)
	}

	delta := autokeep.MakeDeltaDeps(diagnostics, d.imports, d.files, d.knownScopes, d.globalScope)

	toAdd := make(map[string][]string)
	toRemove := make(map[string][]string)

	for _, ruleDeps := range delta.Add {
		targets = append(targets, ruleDeps.Label)
		toAdd[ruleDeps.Label] = append(toAdd[ruleDeps.Label], ruleDeps.Deps...)
	}
	for _, ruleDeps := range delta.Remove {
		targets = append(targets, ruleDeps.Label)
		toRemove[ruleDeps.Label] = append(toRemove[ruleDeps.Label], ruleDeps.Deps...)
	}

	// if no actions could be derived, the build failed and we don't know what to do.
	if len(toAdd) == 0 && len(toRemove) == 0 {
		return fmt.Errorf("build failed, but the corrective action(s) could not be determined.  Manual intervention is required:\n%s", string(out))
	}

	for ruleLabel, deps := range toAdd {
		handler.Add(ruleLabel, collections.SliceDeduplicate(deps))
	}
	for ruleLabel, deps := range toRemove {
		handler.Remove(ruleLabel, collections.SliceDeduplicate(deps))
	}

	if err := handler.Apply(iteration); err != nil {
		return err
	}

	return d.repairIteration(handler, iteration+1)
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

// log.Printf("buildozer 'add deps %s' %s", strings.Join(deps, " "), ruleLabel)
// if err := runBuildozer(
// 	d.progress,
// 	fmt.Sprintf("add deps %s", strings.Join(deps, " ")),
// 	ruleLabel,
// ); err != nil {
// 	return err
// }
// log.Printf("buildozer 'remove deps %s' %s", strings.Join(deps, " "), ruleLabel)
// if err := runBuildozer(
// 	d.progress,
// 	fmt.Sprintf("remove deps %s", strings.Join(deps, " ")),
// 	ruleLabel,
// ); err != nil {
// 	return err
// }

// func runBuildozer(args ...string) error {
// 	// writeBuildProgress(progress, fmt.Sprintf("> buildozer %s", strings.Join(args, " ")))

// 	cmd := exec.Command("buildozer", args...)
// 	cmd.Dir = bazel.GetBuildWorkspaceDirectory()

// 	output, err := cmd.CombinedOutput()
// 	exitCode := procutil.CmdExitCode(cmd, err)

// 	if exitCode != 0 {
// 		return fmt.Errorf("buildozer failed with exit code %d: %s", exitCode, string(output))
// 	}

// 	return nil
// }
