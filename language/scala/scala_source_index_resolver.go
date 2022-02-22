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
	"strings"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/index"
)

func init() {
	CrossResolvers().MustRegisterCrossResolver("stackb:scala-gazelle:scala-source-index", &scalaSourceIndexResolver{
		symbols:    make(map[string]*provider),
		byFilename: make(map[string]*index.ScalaFileSpec),
		byRule:     make(map[label.Label]*index.ScalaRuleSpec),
		parser:     &scalaSourceParser{},
		symbolsMux: &sync.Mutex{},
	})
}

// scalaSourceIndexResolver provides a cross-resolver for scala source files. If
// -scala_source_index_in is configured, the given source index will be used to
// bootstrap the internal cache.  At runtime the .ParseScalaRuleSpec function
// can be used to parse scala files.  If the cache already has an entry for the
// filename with matching sha256, the cache hit will be used.  Otherwise the
// actual parsing will be delegated to the parser backend (a separate process
// that communicates over stdin/stdout).  At the end of gazelle's rule indexing
// phase, .writeIndex is called, dumping the cache into a file (if the outfile
// is configured).  A possible configuration is to use the same file for both in
// and out, creating a configuration loop such that only new/modified .scala
// files need to be parsed on subsequent gazelle executions.
type scalaSourceIndexResolver struct {
	// filesystem path to the indexes to read/write.
	indexIn, indexOut string
	// symbols is a mapping from an import symbol to the thing that provides it.
	symbols map[string]*provider
	// symbolsMux protects symbols map
	symbolsMux *sync.Mutex
	// byFilename is a mapping of the scala file to the spec
	byFilename map[string]*index.ScalaFileSpec
	// byRule is a mapping of the scala rule to the spec
	byRule map[label.Label]*index.ScalaRuleSpec
	// parser is an instance of the scala source parser
	parser *scalaSourceParser
}

type provider struct {
	rule  *index.ScalaRuleSpec
	label label.Label
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaSourceIndexResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.indexIn, "scala_source_index_in", "", "name of the scala source index file to read")
	fs.StringVar(&r.indexOut, "scala_source_index_out", "", "name of the scala source index file to write")
	fs.StringVar(&r.parser.parserToolPath, "scala_parser_tool_path", "sourceindexer", "filesystem path to the parser tool")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaSourceIndexResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexIn != "" {
		if err := r.readScalaRuleIndexSpec(r.indexIn); err != nil {
			log.Println("warning:", err)
		}
	}
	// start the parser backend process
	return r.parser.start()
}

// ParseScalaRuleSpec is used to parse a list of source files.  The list of srcs
// is expected to be relative to the from.Pkg rel field, and the absolute path
// of a file is expected at (dir, from.Pkg, src).  If the resolver already has a
// cache entry for the given file and the current sha256 matches the cached one,
// the cached entry will be returned.  Kind is used to determine if the rule is
// a test rule.
func (r *scalaSourceIndexResolver) ParseScalaRuleSpec(dir string, from label.Label, kind string, srcs ...string) (index.ScalaRuleSpec, error) {
	rule := index.ScalaRuleSpec{
		Label: from.String(),
		Kind:  kind,
		Srcs:  make([]index.ScalaFileSpec, len(srcs)),
	}
	for i, src := range srcs {
		filename := filepath.Join(from.Pkg, src)
		file, err := r.parseScalaFileSpec(dir, filename)
		if err != nil {
			return index.ScalaRuleSpec{}, err
		}
		rule.Srcs[i] = *file
	}
	r.readScalaRuleSpec(rule)
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

	file, err = r.parser.parse(abs)
	if err != nil {
		return nil, fmt.Errorf("scala file parse error %s: %v", abs, err)
	}
	file.Filename = filename
	file.Sha256 = sha256
	r.byFilename[filename] = file
	return file, nil
}

func (r *scalaSourceIndexResolver) readScalaRuleIndexSpec(filename string) error {
	index, err := index.ReadScalaRuleIndexSpec(filename)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", filename, err)
	}

	for _, rule := range index.Rules {
		rCopy := rule
		if err := r.readScalaRuleSpec(rCopy); err != nil {
			return err
		}
	}

	return nil
}

func (r *scalaSourceIndexResolver) readScalaRuleSpec(rule index.ScalaRuleSpec) error {
	ruleLabel, err := label.Parse(rule.Label)
	if err != nil || ruleLabel == label.NoLabel {
		return fmt.Errorf("bad label while loading rule %q: %v", rule.Label, err)
	}

	r.byRule[ruleLabel] = &rule

	for _, file := range rule.Srcs {
		f := &file
		if err := r.readScalaFileSpec(&rule, ruleLabel, f); err != nil {
			return err
		}
	}

	return nil
}

func (r *scalaSourceIndexResolver) readScalaFileSpec(rule *index.ScalaRuleSpec, ruleLabel label.Label, file *index.ScalaFileSpec) error {
	r.symbolsMux.Lock()
	defer r.symbolsMux.Unlock()

	if _, exists := r.byFilename[file.Filename]; exists {
		return fmt.Errorf("duplicate filename: " + file.Filename)
	}

	r.byFilename[file.Filename] = file

	for _, imp := range file.Classes {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Objects {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Traits {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Packages {
		r.providePackage(rule, ruleLabel, file, imp+"._")
	}

	return nil
}

func (r *scalaSourceIndexResolver) provide(rule *index.ScalaRuleSpec, ruleLabel label.Label, file *index.ScalaFileSpec, imp string) {
	log.Println("provide:", imp, ruleLabel)
	if p, ok := r.symbols[imp]; ok {
		if p.label == ruleLabel {
			return
		}
		log.Fatalf("%q is provided by more than one rule (%s, %s)", imp, p.label, ruleLabel)
	}
	r.symbols[imp] = &provider{rule, ruleLabel}
}

func (r *scalaSourceIndexResolver) providePackage(rule *index.ScalaRuleSpec, ruleLabel label.Label, file *index.ScalaFileSpec, imp string) {
	log.Println("provide package:", imp, ruleLabel)
	if p, ok := r.symbols[imp]; ok {
		// if there is an existing provider of the same package for the same rule, that is OK.
		if p.label == ruleLabel {
			return
		}
		// if there is an existing provider of the same package for a different
		// rule, non-test rules take precedence.  If two tests try and provide
		// the same package, the first one wins.
		if isTestRule(rule.Kind) {
			return
		}
		// So the incoming rule.Kind is a not test rule.
		// If two non-test rules try and provide the same package, we have an issue.
		if !isTestRule(p.rule.Kind) {
			log.Printf("current label: %v", p.label)
			log.Printf("next label: %v", ruleLabel)
			if p.rule == rule {
				log.Panicln("huh?")
			}
			log.Printf("current: %+v", p.rule)
			log.Printf("next: %+v", rule)
			log.Fatalf("package %q is provided by more than one rule (%q %s, %q %s)", imp, p.rule.Kind, p.label, rule.Kind, ruleLabel)
		}
	}
	r.symbols[imp] = &provider{rule, ruleLabel}
}

func (r *scalaSourceIndexResolver) OnResolvePhase() error {
	// stop the parser subprocess since the rule indexing phase is over.  No more parsing after this.
	r.parser.stop()

	// dump the index
	return r.writeIndex()
}

func (r *scalaSourceIndexResolver) writeIndex() error {
	// index is not written if the _out file is not configured
	if r.indexOut == "" {
		return nil
	}

	var idx index.ScalaRuleIndexSpec
	for _, rule := range r.byRule {
		idx.Rules = append(idx.Rules, *rule)
	}
	return index.WriteJSONFile(r.indexOut, &idx)
}

// CrossResolve implements the CrossResolver interface.
func (r *scalaSourceIndexResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	log.Println("source crossResolve:", imp.Imp)

	provider, ok := r.symbols[imp.Imp]
	if !ok {
		return nil
	}

	return []resolve.FindResult{{Label: provider.label}}
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

func isTestRule(kind string) bool {
	return strings.Contains(kind, "test")
}
