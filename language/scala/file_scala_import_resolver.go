package scala

import (
	"flag"
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:file", &fileScalaImportResolver{})
}

type fileScalaImportResolver struct {
	// indexFile is the filesystem path to the index.
	indexFile string
	// index is the spec
	index *index.IndexSpec
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *fileScalaImportResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.indexFile, "index_file", "", "name of the index file to read/write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *fileScalaImportResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexFile == "" {
		return nil
	}
	// perform indexing here
	index, err := index.ReadIndexSpec(r.indexFile)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", r.indexFile, err)
	}
	r.index = index
	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *fileScalaImportResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	return nil
}
