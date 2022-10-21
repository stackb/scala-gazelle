package main

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

// notFoundSymbol represents the information collected from parsing the output of
// the scala compiler when a line like
// {{ .Filename }}:{{ .LineNo }}:
// error: Symbol 'type {{ .MissingType }}' is missing from
// the classpath." is found.
type notFoundSymbol struct {
	// From is the bazel label of the affected rule
	From label.Label
	// Filenmame is the name of the affected file
	Filename string
	// Line is the line number of the error
	Line string
	// NotFoundType is the name of the needed type
	NotFoundType string
	// MissingType is the full name of the NotFoundType.
	MissingType string
	// NotFoundFrom is the label that provides the missing type.
	MissingFrom label.Label
}

func (m *notFoundSymbol) ID() string {
	return fmt.Sprintf("nf:filename=%v,type:%v", m.Filename, m.NotFoundType)
}

func (m *notFoundSymbol) Migrate(cfg *config, env ExternalEnvironment) error {
	conflicts := make(map[string]label.Label)

	for imp, from := range cfg.imports {
		suffix := "." + m.NotFoundType
		if !strings.HasSuffix(imp, suffix) {
			continue
		}

		// if .MissingType has already been assigned we have a conflict.
		if m.MissingType != "" {
			conflicts[m.MissingType] = m.MissingFrom
			conflicts[imp] = from
			continue
		}

		m.MissingType = imp
		m.MissingFrom = from
	}

	if len(conflicts) > 0 {
		if err := m.resolveConflicts(cfg, conflicts); err != nil {
			return err
		}
	}

	if m.MissingType == "" {
		if err := m.resolveNoMatch(cfg); err != nil {
			return err
		}
	}

	if m.MissingFrom == label.NoLabel {
		return fmt.Errorf("%s:%s: failed to resolve label for %q (%s)", m.Filename, m.Line, m.MissingType, m.NotFoundType)
	}

	return env.AddMissingDependency(m.MissingType, m.MissingFrom, m.From)
}

func (m *notFoundSymbol) resolveNoMatch(cfg *config) error {
	// try the hints
	if err := m.resolveWithHints(cfg); err != nil {
		return err
	}
	if m.MissingType != "" {
		return nil
	}

	file, ok := cfg.files[m.Filename]
	if !ok {
		return fmt.Errorf("unable to resolve conflicts: source index file %q not found", m.Filename)
	}

	// See if this was one of the imports
	for _, imp := range file.Imports {
		if from, ok := cfg.imports[imp]; ok {
			m.MissingType = imp
			m.MissingFrom = from
			return nil
		}
	}

	// unable to recover
	return fmt.Errorf("%s:%s: failed to resolve concrete type for %q", m.Filename, m.Line, m.NotFoundType)
}

func (m *notFoundSymbol) resolveConflicts(cfg *config, conflicts map[string]label.Label) error {
	// try the hints
	if err := m.resolveWithHints(cfg); err != nil {
		return err
	}
	if m.MissingType != "" {
		return nil
	}

	file, ok := cfg.files[m.Filename]
	if !ok {
		return fmt.Errorf("unable to resolve conflicts: source index file %q not found", m.Filename)
	}

	// Try and find a symbol matching in the current package.
	for _, pkg := range file.Packages {
		withinPkgImp := pkg + "." + m.NotFoundType
		if from, ok := conflicts[withinPkgImp]; ok {
			m.MissingType = withinPkgImp
			m.MissingFrom = from
			return nil
		}
	}

	// See if this was one of the imports
	for _, imp := range file.Imports {
		if from, ok := conflicts[imp]; ok {
			m.MissingType = imp
			m.MissingFrom = from
			return nil
		}
	}

	// unable to resolve conflict.
	imps := make([]string, 0, len(conflicts))
	for imp := range conflicts {
		imps = append(imps, imp)
	}

	return fmt.Errorf("%s:%s: resolved multiple concrete types for %q: %v", m.Filename, m.Line, m.NotFoundType, imps)
}

func (m *notFoundSymbol) resolveWithHints(cfg *config) error {
	hints, ok := cfg.hintMap[m.Filename]
	if !ok {
		return nil
	}

	for _, hint := range hints {
		if hint.symbol != m.NotFoundType {
			continue
		}
		if from, ok := cfg.imports[hint.actual]; ok && from != label.NoLabel {
			m.MissingType = hint.actual
			m.MissingFrom = from
			return nil
		} else {
			return fmt.Errorf("%s:%s: hint suggested to resolve %q to %q but %q is not a known import",
				m.Filename, m.Line, m.NotFoundType, hint.actual, hint.actual)
		}
	}

	return nil
}
