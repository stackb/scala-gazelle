package scala

import (
	"flag"
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:file", &fileScalaImportResolver{})
}

type fileScalaImportResolver struct {
	indexFile string
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (sl *fileScalaImportResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (sl *fileScalaImportResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// CrossResolve implements the CrossResolver interface.
func (sl *scalaLang) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	return nil
}
