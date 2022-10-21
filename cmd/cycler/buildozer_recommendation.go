package main

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/label"
)

// buildozerRecommentation represents the information collected from parsing the progress output of
// a line like ''
type buildozerRecommentation struct {
	// To is the bazel label of the affected rule
	To label.Label
	// Command is the buildozer command
	Command string
}

func (m *buildozerRecommentation) ID() string {
	return fmt.Sprintf("br:command:%v,from:%v", m.Command, m.To)
}

func (m *buildozerRecommentation) Migrate(cfg *config, env ExternalEnvironment) error {
	return env.AddSingleCommand(m.Command, m.To)
}
