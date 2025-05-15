package sweep

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	REPAIRED               = "REPAIRED"
	SweepTransitiveDepsTag = "sweep-transitive-deps"
)

var (
	forRepair                    = make(map[string]*repairSpec)
	postGazelleBuildozerCommands = []string{}
)

func AddPostGazelleBuildozerCommand(action, target string) {
	command := fmt.Sprintf("%s|%s", action, target)
	postGazelleBuildozerCommands = append(postGazelleBuildozerCommands, command)
}

func WritePostGazelleBuildozerFile(filename string) error {
	if len(postGazelleBuildozerCommands) == 0 {
		return nil
	}
	content := strings.Join(postGazelleBuildozerCommands, "\n")
	return os.WriteFile(filename, []byte(content), os.ModePerm)
}

type repairSpec struct {
	// rule is the rule object to be repaired
	rule *rule.Rule
	// from is the label as it would appear in a BUILD file
	from label.Label
	// target is the scala target label that would appear in a log (macro
	// expansion), derived from label rewrites
	target label.Label
}

func MarkForTransitiveRepair(rule *rule.Rule, from label.Label, target label.Label) {
	spec := &repairSpec{rule, from, target}
	forRepair[from.String()] = spec
	forRepair[target.String()] = spec
}

func (d *DepFixer) Transitive() error {
	targets := make([]string, 0, len(forRepair))
	for target := range forRepair {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	for _, target := range targets {
		spec := forRepair[target]

		if hasRepairedComment(spec.rule) {
			log.Printf("%s already repaired (skipping)", target)
		}

		handler := NewTransitiveDeltaDepsHandler(spec)

		if err := d.Repair(handler); err != nil {
			return fmt.Errorf("building rule: %v", err)
		}
	}

	return nil
}

func hasRepairedComment(rule *rule.Rule) bool {
	for _, s := range rule.Comments() {
		if s == REPAIRED {
			return true
		}
	}
	return false
}

func NewTransitiveDeltaDepsHandler(spec *repairSpec) *TransitiveDeltaDepsHandler {
	return &TransitiveDeltaDepsHandler{spec: spec}
}

type TransitiveDeltaDepsHandler struct {
	spec     *repairSpec
	commands []string
}

func (h *TransitiveDeltaDepsHandler) queueCommand(action, target string) {
	command := fmt.Sprintf("%s|%s", action, target)
	log.Println("buildozer:", command)
	h.commands = append(h.commands, command)
}

// Targets implements part of the RepairHandler interface
func (h *TransitiveDeltaDepsHandler) Targets() []string {
	return []string{h.spec.from.String()}
}

// Targets implements part of the RepairHandler interface
func (h *TransitiveDeltaDepsHandler) Add(target string, deps []string) {
	if spec, ok := forRepair[target]; ok {
		target = spec.from.String()
	}
	h.queueCommand("add deps "+strings.Join(deps, " "), target)

	for _, dep := range deps {
		h.queueCommand("comment deps "+dep+" TRANSITIVE", target)
	}
}

// Remove implements part of the RepairHandler interface
func (h *TransitiveDeltaDepsHandler) Remove(target string, deps []string) {
	if spec, ok := forRepair[target]; ok {
		target = spec.from.String()
	}
	h.queueCommand("remove deps "+strings.Join(deps, " "), target)
}

// Done implements part of the RepairHandler interface
func (h *TransitiveDeltaDepsHandler) Done() {
	h.spec.rule.AddComment(REPAIRED)
}

// TarApplygets implements part of the RepairHandler interface
func (h *TransitiveDeltaDepsHandler) Apply(iteration int) error {
	return runBuildozerCommands(h.commands...)
}

func runBuildozerCommands(commands ...string) error {
	// Create the command
	cmd := exec.Command("buildozer", "-f", "-")

	// Create a pipe to send input to the command
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %v", err)
	}

	// Create buffers to capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %v", err)
	}

	// Send input to the command
	input := strings.Join(commands, "\n")
	log.Println("buildozer input:\n", input)
	_, err = io.WriteString(stdin, input)
	if err != nil {
		return fmt.Errorf("writing to stdin: %v", err)
	}

	// Close stdin to signal that we're done sending input
	stdin.Close()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("waiting for command (%v): stderr: %s", err, stderr.String())
	}

	// Print the output
	return nil
}

func HasSweepTransitiveDepsTag(r *rule.Rule) bool {
	for _, tag := range r.AttrStrings("tags") {
		if tag == SweepTransitiveDepsTag {
			return true
		}
	}
	return false
}

func RemoveSweepTransitiveDepsTag(r *rule.Rule) {
	tags := make([]string, 0)

	for _, tag := range r.AttrStrings("tags") {
		if tag == SweepTransitiveDepsTag {
			continue
		}
		tags = append(tags, tag)
	}

	if len(tags) == 0 {
		r.DelAttr("tags")
	} else {
		r.SetAttr("tags", tags)
	}
}
