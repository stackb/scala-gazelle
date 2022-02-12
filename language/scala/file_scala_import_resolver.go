package scala

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:file", &fileScalaImportResolver{})
}

type fileScalaImportResolver struct {
	// index is a flag that, if true, instructs the resolver to perform indexing
	// and write the indexFile.  Otherwise, only read the file.
	index bool
	// indexFile is the filesystem path to the index.
	indexFile string
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *fileScalaImportResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.BoolVar(&r.index, "index", false, "if true, perform indexing")
	fs.StringVar(&r.indexFile, "index_file", "", "name of the index file to read/write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *fileScalaImportResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	// perform indexing here
	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *fileScalaImportResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	return nil
}
