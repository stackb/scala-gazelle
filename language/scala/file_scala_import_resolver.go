package scala

import (
	"flag"
	"fmt"
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:file", &fileScalaImportResolver{
		byLabel: make(map[string][]label.Label),
	})
}

// fileScalaImportResolver provides a cross-resolver for precompiled symbols that are
// provided by the mergeindex tool.
type fileScalaImportResolver struct {
	// indexFile is the filesystem path to the index.
	indexFile string
	// byLabel is a mapping from an import string to the label that provides it.
	// It is possible more than one label provides a class.
	byLabel map[string][]label.Label
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

	for _, jarSpec := range index.JarSpecs {
		for _, class := range jarSpec.Classes {
			jarLabel, err := label.Parse(jarSpec.Label)
			if err != nil {
				log.Println("bad label while loading jar spec %s: %v", jarSpec.Filename, err)
				continue
			}
			r.byLabel[class] = append(r.byLabel[class], jarLabel)
		}
	}

	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *fileScalaImportResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	if lang != "scala" {
		return nil
	}

	resolved := r.byLabel[imp.Imp]
	if len(resolved) == 0 {
		return nil
	}

	result := make([]resolve.FindResult, len(resolved))
	for i, v := range resolved {
		result[i] = resolve.FindResult{Label: v}
	}

	return result
}
