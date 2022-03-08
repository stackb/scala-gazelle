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

type ScalaFileParser interface {
	// ParseScalaFiles is used to parse a list of source files.  The list of srcs
	// is expected to be relative to the from.Pkg rel field, and the absolute path
	// of a file is expected at (dir, from.Pkg, src).  If the resolver already has a
	// cache entry for the given file and the current sha256 matches the cached one,
	// the cached entry will be returned.  Kind is used to determine if the rule is
	// a test rule.
	ParseScalaFiles(dir string, from label.Label, kind string, srcs ...string) (index.ScalaRuleSpec, error)
}

// ScalaRuleRegistry keep track of which files are associated under a rule
// (which has a srcs attribute).
type ScalaRuleRegistry interface {
	// GetScalaFiles returns the rule spec for a given label.  If the label is unknown, false is returned.
	GetScalaRule(from label.Label) (*index.ScalaRuleSpec, bool)
}

func newScalaSourceIndexResolver() *scalaSourceIndexResolver {
	return &scalaSourceIndexResolver{
		providers:    make(map[string][]*provider),
		packages:     make(map[string][]*provider),
		byFilename:   make(map[string]*index.ScalaFileSpec),
		byRule:       make(map[label.Label]*index.ScalaRuleSpec),
		parser:       &scalaSourceParser{},
		providersMux: &sync.Mutex{},
	}
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
	// providers and packages is a mapping from an import symbol to the things
	// that provide it. It is legal for more than one label to provide a symbol
	// (e.g., a test class can exist in multiple rule srcs attribute), but it is
	// an error if such a symbol is attempted to be imported (e.g., a test class
	// should not be imported). They are made distinct as they have different
	// disambigation semantics.
	providers map[string][]*provider
	packages  map[string][]*provider
	// providersMux protects providers map
	providersMux *sync.Mutex
	// byFilename is a mapping of the scala file to the spec
	byFilename map[string]*index.ScalaFileSpec
	// byRule is a mapping of the scala rule to the spec
	byRule map[label.Label]*index.ScalaRuleSpec
	// parser is an instance of the scala source parser
	parser *scalaSourceParser
}

type provider struct {
	rule  *index.ScalaRuleSpec
	file  *index.ScalaFileSpec
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

// ParseScalaFiles implements ScalaFileParser
func (r *scalaSourceIndexResolver) ParseScalaFiles(dir string, from label.Label, kind string, srcs ...string) (index.ScalaRuleSpec, error) {
	rule := &index.ScalaRuleSpec{
		Label: from.String(),
		Kind:  kind,
		Srcs:  make([]*index.ScalaFileSpec, len(srcs)),
	}
	for i, src := range srcs {
		filename := filepath.Join(from.Pkg, src)
		file, err := r.parseScalaFileSpec(dir, filename)
		if err != nil {
			return index.ScalaRuleSpec{}, err
		}
		rule.Srcs[i] = file
	}
	r.readScalaRuleSpec(rule)
	return *rule, nil
}

// GetScalaRule implements ScalaRuleRegistry.
func (r *scalaSourceIndexResolver) GetScalaRule(from label.Label) (*index.ScalaRuleSpec, bool) {
	from.Repo = "" // TODO(pcj): this is correct?  We always want sources in the main repo.
	rule, ok := r.byRule[from]
	return rule, ok
}

// Provided implements the protoc.ImportProvider interface.
func (r *scalaSourceIndexResolver) Provided(lang, impLang string) map[label.Label][]string {
	if lang != "scala" && impLang != "scala" {
		return nil
	}

	result := make(map[label.Label][]string)
	for imp, pp := range r.providers {
		for _, p := range pp {
			result[p.label] = append(result[p.label], imp)
		}
	}

	return result
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
			// log.Printf("file cache hit: <%s> (%s)", filename, sha256)
			return file, nil
		} else {
			log.Printf("sha256 mismatch: <%s> (%s, %s)", filename, file.Sha256, sha256)
		}
	} else {
		// log.Printf("file cache miss: <%s>", filename)
	}

	file, err = r.parser.parse(abs)
	if err != nil {
		return nil, fmt.Errorf("scala file parse error %s: %v", abs, err)
	}
	file.Filename = filename
	file.Sha256 = sha256
	log.Printf("Parsed: <%s>", filename)
	return file, nil
}

func (r *scalaSourceIndexResolver) readScalaRuleIndexSpec(filename string) error {
	index, err := index.ReadScalaRuleIndexSpec(filename)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", filename, err)
	}

	for _, rule := range index.Rules {
		if err := r.readScalaRuleSpec(rule); err != nil {
			return err
		}
	}

	return nil
}

func (r *scalaSourceIndexResolver) readScalaRuleSpec(rule *index.ScalaRuleSpec) error {
	ruleLabel, err := label.Parse(rule.Label)
	if err != nil || ruleLabel == label.NoLabel {
		return fmt.Errorf("bad label while loading rule %q: %v", rule.Label, err)
	}

	for _, file := range rule.Srcs {
		if err := r.readScalaFileSpec(rule, ruleLabel, file); err != nil {
			return err
		}
	}

	r.byRule[ruleLabel] = rule

	return nil
}

func (r *scalaSourceIndexResolver) readScalaFileSpec(rule *index.ScalaRuleSpec, ruleLabel label.Label, file *index.ScalaFileSpec) error {
	r.providersMux.Lock()
	defer r.providersMux.Unlock()

	if _, exists := r.byFilename[file.Filename]; exists {
		return fmt.Errorf("duplicate filename <%s>", file.Filename)
	}

	for _, imp := range file.Classes {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Objects {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Traits {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Types {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Vals {
		r.provide(rule, ruleLabel, file, imp)
	}
	for _, imp := range file.Packages {
		r.providePackage(rule, ruleLabel, file, imp)
	}

	r.byFilename[file.Filename] = file
	// log.Printf("cached file <%s> (%s) %+v", file.Filename, file.Sha256, file)

	return nil
}

func (r *scalaSourceIndexResolver) provide(rule *index.ScalaRuleSpec, ruleLabel label.Label, file *index.ScalaFileSpec, imp string) {
	if pp, ok := r.providers[imp]; ok {
		p := pp[0]
		if p.label == ruleLabel {
			return
		}
		log.Printf("%q is provided by more than one rule (%s, %s)", imp, p.label, ruleLabel)
	}
	r.providers[imp] = append(r.providers[imp], &provider{rule, file, ruleLabel})
}

func (r *scalaSourceIndexResolver) providePackage(rule *index.ScalaRuleSpec, ruleLabel label.Label, file *index.ScalaFileSpec, imp string) {
	if pp, ok := r.packages[imp]; ok {
		p := pp[0]
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
	}
	r.packages[imp] = append(r.packages[imp], &provider{rule, file, ruleLabel})
}

// OnResolvePhase implements GazellePhaseTransitionListener.
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
		idx.Rules = append(idx.Rules, rule)
	}

	if err := index.WriteJSONFile(r.indexOut, &idx); err != nil {
		return err
	}

	log.Println("Wrote", r.indexOut)

	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *scalaSourceIndexResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	if lang != "scala" {
		return nil
	}

	sym := imp.Imp

	if providers, ok := r.providers[sym]; ok {
		result := make([]resolve.FindResult, len(providers))
		for i, p := range providers {
			log.Printf("source crossResolve %q provider hit %d: %v", imp.Imp, i, p.label)
			result[i] = resolve.FindResult{Label: p.label}
		}
		return result
	}

	sym = strings.TrimSuffix(sym, "._")

	if packages, ok := r.packages[sym]; ok {
		// pick the first result -- this might not be correct!
		result := make([]resolve.FindResult, len(packages))
		for i, p := range packages {
			log.Printf("source crossResolve %q package hit %d: %v", imp.Imp, i, p.label)
			result[i] = resolve.FindResult{Label: p.label}
		}
		return result
	}

	// log.Println("source crossResolve miss:", imp.Imp)
	return nil
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
