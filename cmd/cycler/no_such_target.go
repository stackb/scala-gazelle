package main

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"
)

// noSuchTarget represents the information collected from parsing the output of
// the scala compiler when a line like
// {{ .Filename }}:{{ .LineNo }}:
// error: Symbol 'type {{ .MissingType }}' is missing from
// the classpath." is found.
type noSuchTarget struct {
	// From is the bazel label of the affected rule
	From label.Label
	// Filenmame is the name of the affected BUILD file
	Filename string
	// Line is the line number of the error
	Line string
	// Column is the column number of the error
	Column string
	// Dep is the dependency label that should be removed.
	Dep label.Label
}

func (m *noSuchTarget) ID() string {
	return fmt.Sprintf("nst:filename=%v,dep:%v,from:%v", m.Filename, m.Dep, m.From)
}

func (m *noSuchTarget) Migrate(cfg *config, env ExternalEnvironment) error {
	return env.RemoveDependency(m.Dep, m.From)
}
