package scala

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/index"
)

// PlatformLabel represents a label that does not need to be included in deps.
// Example: 'java.lang.Boolean'.
var PlatformLabel = label.New("platform", "", "do_not_import")

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:scala-class-index", &scalaClassIndexResolver{
		byLabel: make(map[string][]label.Label),
	})
}

// scalaClassIndexResolver provides a cross-resolver for symbols extracted from
// jar files.  If -scala_class_index_file is configured, the internal cache will
// be bootstrapped with the contents of that file (typically the `java_index`
// rule is used to generate it).  JarSpec entries that have an empty label are
// assigned a special label 'PlatformLabel' which means ("you don't need to add
// anything to deps for this import, it's implied by the platform").  Typically
// platform implied jars are specified in the `java_index.platform_jars`
// attribute. At runtime, the cache is used to resolve scala import symbols
// during the gazelle dependency resolution phase. If a query for
// 'com.google.gson.Gson' yields '@maven//:com_google_code_gson_gson', that
// value should be added to deps.  If a query for 'java.lang.Boolean' yields the
// PlatformLabel, it can be skipped.
type scalaClassIndexResolver struct {
	// indexIn is the filesystem path to the index.
	indexIn string
	// byLabel is a mapping from an import string to the label that provides it.
	// It is possible more than one label provides a class.
	byLabel map[string][]label.Label
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaClassIndexResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.indexIn, "scala_class_index_file", "", "name of the scala class index file to read/write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaClassIndexResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexIn == "" {
		return nil
	}
	// perform indexing here
	index, err := index.ReadIndexSpec(r.indexIn)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", r.indexIn, err)
	}

	for _, jarSpec := range index.JarSpecs {
		jarLabel, err := label.Parse(jarSpec.Label)
		if err != nil {
			if jarSpec.Label == "" {
				jarLabel = PlatformLabel
			} else {
				log.Fatalf("bad label while loading jar spec %s: %v", jarSpec.Filename, err)
				continue
			}
		}
		for _, pkg := range jarSpec.Packages {
			r.byLabel[pkg] = append(r.byLabel[pkg], jarLabel)
		}

		for _, class := range jarSpec.Classes {
			r.byLabel[class] = append(r.byLabel[class], jarLabel)
		}
	}

	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *scalaClassIndexResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	if lang != "scala" {
		return nil
	}

	sym := strings.TrimSuffix(imp.Imp, "._")

	resolved := r.byLabel[sym]
	if len(resolved) == 0 {
		return nil
	}

	result := make([]resolve.FindResult, len(resolved))
	for i, v := range resolved {
		result[i] = resolve.FindResult{Label: v}
	}

	return result
}
