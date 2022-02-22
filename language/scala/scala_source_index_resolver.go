package scala

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
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
		byRule:     make(map[label.Label]*index.ScalaRuleSpec),
		parser:     &scalaSourceParser{},
	})
}

// scalaSourceIndexResolver provides a cross-resolver for precompiled symbols
// that are provided by the mergeindex tool.
type scalaSourceIndexResolver struct {
	// filesystem path to the indexes to read/write.
	indexIn, indexOut string
	// byLabel is a mapping from an import symbol to the label that provides it.
	// It is possible more than one label provides a symbol.
	byLabel map[string][]label.Label
	// byFilename is a mapping of the scala file to the spec
	byFilename map[string]*index.ScalaFileSpec
	// byRule is a mapping of the scala rule to the spec
	byRule map[label.Label]*index.ScalaRuleSpec
	// parser is an instance of the scala source parser
	parser *scalaSourceParser
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaSourceIndexResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	log.Println("os.Args", os.Args)
	fs.StringVar(&r.indexIn, "scala_source_index_in", "", "name of the scala source index file to read")
	fs.StringVar(&r.indexOut, "scala_source_index_out", "", "name of the scala source index file to write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaSourceIndexResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexIn != "" {
		if err := r.ReadIndex(r.indexIn); err != nil {
			log.Println("warning:", err)
		}
	}
	return nil
}

func (r *scalaSourceIndexResolver) ParseScalaRuleSpec(dir string, from label.Label, srcs ...string) (*index.ScalaRuleSpec, error) {
	rule := &index.ScalaRuleSpec{
		Label: from.String(),
		Srcs:  make([]index.ScalaFileSpec, len(srcs)),
	}
	for i, src := range srcs {
		filename := filepath.Join(from.Pkg, src)
		file, err := r.parseScalaFileSpec(dir, filename)
		if err != nil {
			return nil, err
		}
		rule.Srcs[i] = *file
	}
	r.byRule[from] = rule
	return rule, nil
}

func (r *scalaSourceIndexResolver) parseScalaFileSpec(dir, filename string) (*index.ScalaFileSpec, error) {
	abs := filepath.Join(dir, filename)
	sha256, err := fileSha256(abs)
	if err != nil {
		return nil, fmt.Errorf("scala file sha256 error %s: %v", abs, err)
	}

	file, ok := r.byFilename[filename]
	if ok {
		if file.Sha256 == sha256 {
			return file, nil
		}
	}

	log.Println("parsing ->", filename)
	file, err = r.parser.parse(abs)
	if err != nil {
		return nil, fmt.Errorf("scala file parse error %s: %v", abs, err)
	}
	file.Filename = filename
	file.Sha256 = sha256
	r.byFilename[filename] = file
	return file, nil
}

func (r *scalaSourceIndexResolver) ReadIndex(filename string) error {
	log.Println("Reading source index", filename)
	index, err := index.ReadScalaRuleIndexSpec(filename)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", filename, err)
	}

	resolver := protoc.GlobalResolver()
	lang := "scala"

	for _, rule := range index.Rules {
		ruleLabel, err := label.Parse(rule.Label)
		if err != nil {
			log.Printf("bad label while loading rule spec: %v", err)
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
		r.byRule[ruleLabel] = &rule
	}

	return nil
}

func (r *scalaSourceIndexResolver) WriteIndex() error {
	// index is not written if the _out file is not configured
	if r.indexOut == "" {
		return nil
	}
	log.Println("Writing source index", r.indexOut)

	var idx index.ScalaRuleIndexSpec
	for _, rule := range r.byRule {
		idx.Rules = append(idx.Rules, *rule)
	}
	return index.WriteJSONFile(r.indexOut, &idx)
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

// fileSha256 computes the sha256 hash of a file
func fileSha256(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return readSha256(f)
}

// Compute the sha256 hash of a reader
func readSha256(in io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, in); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
