package scala

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/stackb/rules_proto/pkg/protoc"

	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:scala-source-index", &scalaSourceIndexResolver{
		byLabel:    make(map[string][]label.Label),
		byFilename: make(map[string]*index.ScalaFileSpec),
		parser:     &scalaSourceParser{},
	})
}

// scalaSourceIndexResolver provides a cross-resolver for precompiled symbols
// that are provided by the mergeindex tool.
type scalaSourceIndexResolver struct {
	// indexFile is the filesystem path to the index.
	indexFile string
	// byLabel is a mapping from an import symbol to the label that provides it.
	// It is possible more than one label provides a symbol.
	byLabel map[string][]label.Label
	// byFilename is a mapping of the scala file to the spec
	byFilename map[string]*index.ScalaFileSpec
	// parser is
	parser *scalaSourceParser
}

func (r *scalaSourceIndexResolver) LookupScalaFileSpec(filename string) (*index.ScalaFileSpec, bool) {
	file, ok := r.byFilename[filename]
	return file, ok
}

func (r *scalaSourceIndexResolver) ParseScalaFileSpec(dir, filename string) (*index.ScalaFileSpec, error) {
	file, ok := r.LookupScalaFileSpec(filename)
	if ok {
		return file, nil
	}
	abs := filepath.Join(dir, filename)
	log.Println("parsing ->", filename)
	file, err := r.parser.parse(abs)
	if err != nil {
		return nil, fmt.Errorf("scala file parse error %s: %v", abs, err)
	}
	file.Filename = filename
	r.byFilename[filename] = file
	return file, nil
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaSourceIndexResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.indexFile, "scala_source_index_file", "", "name of the scala source index file to read/write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaSourceIndexResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexFile == "" {
		return nil
	}

	index, err := index.ReadScalaRuleIndexSpec(r.indexFile)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", r.indexFile, err)
	}

	resolver := protoc.GlobalResolver()
	lang := "scala"

	for _, rule := range index.Rules {
		ruleLabel, err := label.Parse(rule.Label)
		if err != nil {
			log.Println("bad label while loading rule spec: %v", err)
			continue
		}

		for _, file := range rule.Srcs {
			f := file
			if _, exists := r.byFilename[f.Filename]; exists {
				panic("duplicate filename: " + f.Filename)
			}
			r.byFilename[f.Filename] = &f

			for _, sym := range f.Classes {
				resolver.Provide(lang, lang, sym, ruleLabel)
				// r.byLabel[sym] = append(r.byLabel[sym], ruleLabel)
			}
			for _, sym := range f.Objects {
				resolver.Provide(lang, lang, sym, ruleLabel)
				// r.byLabel[sym] = append(r.byLabel[sym], ruleLabel)
			}
			for _, sym := range f.Traits {
				resolver.Provide(lang, lang, sym, ruleLabel)
				// r.byLabel[sym] = append(r.byLabel[sym], ruleLabel)
			}
			for _, sym := range f.Packages {
				resolver.Provide(lang, lang, sym+"._", ruleLabel)
				// r.byLabel[sym] = append(r.byLabel[sym], ruleLabel)
			}
		}
	}

	return nil
}

func (r *scalaSourceIndexResolver) DumpIndex(filename string) error {
	var idx index.ScalaRuleSpec
	for _, v := range r.byFilename {
		idx.Srcs = append(idx.Srcs, *v)
	}
	return index.WriteJSONFile(filename, &idx)
}

// CrossResolve implements the CrossResolver interface.
func (r *scalaSourceIndexResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
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
