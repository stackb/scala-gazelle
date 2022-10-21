package main

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

// missingSymbol represents the information collected from parsing the output of
// the scala compiler when a line like
// {{ .Filename }}:{{ .LineNo }}:
// error: Symbol 'type {{ .MissingType }}' is missing from
// the classpath." is found.
type missingSymbol struct {
	// From is the bazel label of the affected rule
	From label.Label
	// Filenmame is the name of the affected file
	Filename string
	// Line is the line number of the error
	Line string
	// MissingType is the name of the needed type
	MissingType string
	// RequiredType is the name of the type that missing requires.
	RequiredType string
	// RequiredFrom is the label that provides the required type.
	RequiredFrom label.Label
	// MissingFrom is the label that provides the missing type.
	MissingFrom label.Label
}

func (m *missingSymbol) ID() string {
	return fmt.Sprintf("ms:filename=%v,missingType=%v,requiredType=%v", m.Filename, m.MissingType, m.RequiredType)
}

func (m *missingSymbol) Migrate(cfg *config, env ExternalEnvironment) error {
	if imp, from, err := matchImport(cfg.imports, cfg.symbolMappings, m.MissingType); err != nil {
		return fmt.Errorf("failed to match import for .MissingType %+v: %v", m, err)
	} else {
		if to, ok := cfg.labelMappings[from]; ok {
			from = to
		}
		m.MissingType = imp
		m.MissingFrom = from
	}

	if imp, from, err := matchImport(cfg.imports, cfg.symbolMappings, m.RequiredType); err != nil {
		return fmt.Errorf("failed to match import for .RequiredType %+v: %v", m, err)
	} else {
		if to, ok := cfg.labelMappings[from]; ok {
			from = to
		}
		m.RequiredType = imp
		m.RequiredFrom = from
	}

	if m.MissingFrom == label.NoLabel {
		return fmt.Errorf("%s:%s: failed to resolve label for %q (%s)", m.Filename, m.Line, m.RequiredType, m.MissingType)
	}

	return env.AddMissingDependency(m.MissingType, m.MissingFrom, m.From)
}

func parentTypeName(name string) (string, bool) {
	lastDot := strings.LastIndex(name, ".")
	if lastDot <= 0 {
		return "", false
	}
	pkg := name[0:lastDot]
	return pkg, true
}

func matchImport(imports map[string]label.Label, symbolMappings map[string]string, imp string) (string, label.Label, error) {
	current := imp
	for {
		if from, ok := imports[current]; ok {
			return imp, from, nil
		}
		if alt, ok := symbolMappings[current]; ok {
			if from, ok := imports[alt]; ok {
				return alt, from, nil
			}
		}
		if parent, ok := parentTypeName(current); !ok {
			return "", label.NoLabel, fmt.Errorf("failed to match import %q", imp)
		} else {
			current = parent
		}
	}
}
