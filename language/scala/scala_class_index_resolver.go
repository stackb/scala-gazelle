package scala

import (
	"flag"
	"fmt"
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/stackb/rules_proto/pkg/protoc"

	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:scala-class-index", &scalaClassIndexResolver{
		byLabel: make(map[string][]label.Label),
	})
}

// scalaClassIndexResolver provides a cross-resolver for precompiled symbols that are
// provided by the mergeindex tool.
type scalaClassIndexResolver struct {
	// indexFile is the filesystem path to the index.
	indexFile string
	// byLabel is a mapping from an import string to the label that provides it.
	// It is possible more than one label provides a class.
	byLabel map[string][]label.Label
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaClassIndexResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.indexFile, "scala_class_index_file", "", "name of the scala class index file to read/write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaClassIndexResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexFile == "" {
		return nil
	}
	// perform indexing here
	index, err := index.ReadIndexSpec(r.indexFile)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", r.indexFile, err)
	}

	resolver := protoc.GlobalResolver()
	lang := "scala"

	for _, jarSpec := range index.JarSpecs {
		jarLabel, err := label.Parse(jarSpec.Label)
		if err != nil {
			log.Printf("bad label while loading jar spec %s: %v", jarSpec.Filename, err)
			continue
		}
		for _, pkg := range jarSpec.Packages {
			pkgImport := pkg + "._"
			r.byLabel[pkgImport] = append(r.byLabel[pkgImport], jarLabel)
			resolver.Provide(lang, lang, pkgImport, jarLabel)

		}

		for _, class := range jarSpec.Classes {
			r.byLabel[class] = append(r.byLabel[class], jarLabel)
			resolver.Provide(lang, lang, class, jarLabel)

		}
	}

	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *scalaClassIndexResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
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
