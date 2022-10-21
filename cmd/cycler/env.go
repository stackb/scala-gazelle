package main

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

const onlyProtoMigrationMode = true

// PlatformLabel represents a label that does not need to be included in deps.
// Example: 'java.lang.Boolean'. TODO: deduplicate this from the one is language/scala.
var PlatformLabel = label.New("platform", "", "do_not_import")

// ExternalEnvironment is an interface to make changes to the BUILD graph (via buildozer).
type ExternalEnvironment interface {
	AddSingleCommand(command string, to label.Label) error
	RemoveDependency(dep, from label.Label) error
	AddMissingDependency(imp string, dep, to label.Label) error
}

// Migration is is capable to effecting change to a proscribed external environment.
type Migration interface {
	ID() string
	Migrate(cfg *config, external ExternalEnvironment) error
}

// env is a container for things that are gathered during each cycle of the program.
type env struct {
	cfg *config
	// errs are errors reported during a run.
	errs []error
	// commands tracks the number of changes we make to external environment.
	// The boolean is whether the command was actually written to the file.
	commands map[string]bool
}

func newEnv(cfg *config) *env {
	return &env{
		cfg:      cfg,
		errs:     make([]error, 0),
		commands: make(map[string]bool),
	}
}

func (e *env) run() ([]string, []error) {
	events, err := readBuildEvents(e.cfg.buildEventsFile)
	if err != nil {
		return nil, []error{err}
	}

	var migrations []Migration

	for _, evt := range events {
		if evt.Action != nil {
			if evt.Action.ExitCode == 0 {
				continue
			}
			if evt.Action.Stderr.Name == "" {
				continue
			}

			mm, err := evt.Action.parseStderr(e)
			if err != nil {
				e.ReportError(err)
			} else {
				migrations = append(migrations, mm...)
			}
		} else if evt.Progress != nil {
			if evt.Progress.Stderr == "" {
				continue
			}
			mm, err := evt.Progress.parseStderr(e)
			if err != nil {
				e.ReportError(err)
			} else {
				migrations = append(migrations, mm...)
			}
		}
	}

	for _, m := range migrations {
		if err := m.Migrate(e.cfg, e); err != nil {
			e.ReportError(err)
			continue
		}
	}

	hasCommand := make(map[string]bool)
	var newCommands []string

	for _, cmd := range e.cfg.buildozerCommands {
		if hasCommand[cmd] {
			continue
		}
		hasCommand[cmd] = true
	}

	for cmd := range e.commands {
		if hasCommand[cmd] {
			// log.Println("SKIP (already exists):", cmd)
			continue
		}
		newCommands = append(newCommands, cmd)
	}

	return newCommands, e.errs
}

func (e *env) ReportError(err error) {
	e.errs = append(e.errs, err)
}

func (e *env) RemoveDependency(dep, from label.Label) error {
	if dep == label.NoLabel {
		return nil
	}
	if actual, ok := e.cfg.labelMappings[dep]; ok {
		dep = actual
	}
	if actual, ok := e.cfg.labelMappings[from]; ok {
		from = actual
	}

	if err := e.addCommand(fmt.Sprintf("remove deps %v|%v", dep, from)); err != nil {
		return err
	}

	return nil
}

func (e *env) AddSingleCommand(command string, to label.Label) error {
	if actual, ok := e.cfg.labelMappings[to]; ok {
		to = actual
	}

	if err := e.addCommand(fmt.Sprintf("%v|%v", command, to)); err != nil {
		return err
	}

	return nil
}

func (e *env) AddMissingDependency(imp string, dep, to label.Label) error {
	if dep == label.NoLabel {
		return nil
	}
	if dep == PlatformLabel {
		return nil
	}

	if actual, ok := e.cfg.labelMappings[dep]; ok {
		dep = actual
	}
	if actual, ok := e.cfg.labelMappings[to]; ok {
		to = actual
	}

	if onlyProtoMigrationMode {
		if !strings.HasSuffix(dep.Name, "_scala_library") {
			return fmt.Errorf("skipped missing dep (does not look like a proto dep and onlyProtoMigrationMode=true): %q %v|%v", imp, dep, to)
		}
	}

	if false {
		if err := e.addCommand(fmt.Sprintf("comment deps scala-import:\\ %s|%v", imp, to)); err != nil {
			return err
		}
	}

	if err := e.addCommand(fmt.Sprintf("add deps %v|%v", dep, to)); err != nil {
		return err
	}

	return nil
}

func (e *env) addCommand(cmd string) error {
	if _, ok := e.commands[cmd]; ok {
		return nil
	}
	e.commands[cmd] = false
	return nil
}
