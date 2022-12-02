package crossresolve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/scalaparse"
)

func NewScalaSourceCrossResolver(lang string) *ScalaSourceCrossResolver {
	return &ScalaSourceCrossResolver{
		lang:         lang,
		parser:       scalaparse.NewScalaParseServer(),
		providersMux: &sync.Mutex{},
		providers:    make(map[string][]*providerSpec),
		packages:     make(map[string][]*providerSpec),
		byFilename:   make(map[string]*sppb.File),
		byRule:       make(map[label.Label]*sppb.Rule),
	}
}

// ScalaSourceCrossResolver provides a cross-resolver for scala source files. If
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
type ScalaSourceCrossResolver struct {
	// lang is the language name cross resolution should match on, typically "scala".
	lang string
	// providers and packages is a mapping from an import symbol to the things
	// that provide it. It is legal for more than one label to provide a symbol
	// (e.g., a test class can exist in multiple rule srcs attribute), but it is
	// an error if such a symbol is attempted to be imported (e.g., a test class
	// should not be imported). They are made distinct as they have different
	// disambigation semantics.
	providers map[string][]*providerSpec
	packages  map[string][]*providerSpec
	// providersMux protects providers map
	providersMux *sync.Mutex
	// byFilename is a mapping of the scala file to the spec
	byFilename map[string]*sppb.File
	// byRule is a mapping of the scala rule to the spec
	byRule map[label.Label]*sppb.Rule
	// parser is an instance of the scala source parser
	parser *scalaparse.ScalaParseServer
}

type providerSpec struct {
	rule  *sppb.Rule
	file  *sppb.File
	label label.Label
}

// Start begins the parser process
func (r *ScalaSourceCrossResolver) Start() error {
	if err := r.parser.Start(); err != nil {
		return fmt.Errorf("starting parser: %w", err)
	}
	return nil
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *ScalaSourceCrossResolver) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *ScalaSourceCrossResolver) CheckFlags(flags *flag.FlagSet, c *config.Config) error {
	return nil
}

// ParseScalaFiles implements scalaparse.ScalaParser
func (r *ScalaSourceCrossResolver) ParseScalaFiles(from label.Label, kind string, dir string, srcs ...string) (*sppb.Rule, error) {
	rule := &sppb.Rule{
		Label: from.String(),
		Kind:  kind,
		Files: make([]*sppb.File, len(srcs)),
	}
	for i, src := range srcs {
		filename := filepath.Join(from.Pkg, src)
		file, err := r.parseScalaFileIndex(dir, filename)
		if err != nil {
			return nil, err
		}
		rule.Files[i] = file
	}
	if err := r.AddRule(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// Provided implements the protoc.ImportProvider interface.
func (r *ScalaSourceCrossResolver) Provided(lang, impLang string) map[label.Label][]string {
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

func (r *ScalaSourceCrossResolver) parseScalaFileIndex(dir, filename string) (*sppb.File, error) {
	t1 := time.Now()

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
			// log.Printf("sha256 mismatch: <%s> (%s, %s)", filename, file.Sha256, sha256)
		}
	} else {
		// log.Printf("file cache miss: <%s>", filename)
	}

	response, err := r.parser.Parse(context.Background(), &sppb.ParseRequest{
		Filenames: []string{abs},
	})
	if err != nil {
		return nil, fmt.Errorf("scala file parse error %s: %v", abs, err)
	}

	t2 := time.Now().Sub(t1).Round(1 * time.Millisecond)

	if response.Error != "" {
		log.Printf("Parse Error <%s>: %s", filename, response.Error)
	} else {
		log.Printf("Parsed <%s> (%v)", filename, t2)
	}

	scalaFile := response.Files[0]
	return &sppb.File{
		Filename: filename,
		Packages: scalaFile.Packages,
		Imports:  scalaFile.Imports,
		Classes:  scalaFile.Classes,
		Types:    scalaFile.Types,
		Vals:     scalaFile.Vals,
		Objects:  scalaFile.Objects,
		Traits:   scalaFile.Traits,
		Sha256:   sha256,
	}, nil
}

func (r *ScalaSourceCrossResolver) Rules() []*sppb.Rule {
	rules := make([]*sppb.Rule, 0, len(r.byRule))
	for _, r := range r.byRule {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool {
		a := rules[i]
		b := rules[j]
		return a.Label < b.Label
	})
	return rules
}

func (r *ScalaSourceCrossResolver) AddRule(rule *sppb.Rule) error {
	ruleLabel, err := label.Parse(rule.Label)
	if err != nil || ruleLabel == label.NoLabel {
		return fmt.Errorf("bad label while loading rule %q: %v", rule.Label, err)
	}

	for _, file := range rule.Files {
		if err := r.readScalaFile(rule, ruleLabel, file); err != nil {
			return err
		}
	}

	r.byRule[ruleLabel] = rule

	return nil
}

func (r *ScalaSourceCrossResolver) readScalaFile(rule *sppb.Rule, ruleLabel label.Label, file *sppb.File) error {
	r.providersMux.Lock()
	defer r.providersMux.Unlock()

	if _, exists := r.byFilename[file.Filename]; exists {
		// return fmt.Errorf("duplicate filename <%s>", file.Filename)
		return nil
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

func (r *ScalaSourceCrossResolver) provide(rule *sppb.Rule, ruleLabel label.Label, file *sppb.File, imp string) {
	if pp, ok := r.providers[imp]; ok {
		p := pp[0]
		if p.label == ruleLabel {
			return
		}
		log.Printf("%q is provided by more than one rule (%s, %s)", imp, p.label, ruleLabel)
	}
	log.Printf("%q is provided by %s", imp, ruleLabel)
	r.providers[imp] = append(r.providers[imp], &providerSpec{rule, file, ruleLabel})
}

func (r *ScalaSourceCrossResolver) providePackage(rule *sppb.Rule, ruleLabel label.Label, file *sppb.File, imp string) {
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
	r.packages[imp] = append(r.packages[imp], &providerSpec{rule, file, ruleLabel})
}

// OnResolve implements GazellePhaseTransitionListener.
func (r *ScalaSourceCrossResolver) OnResolve() {
	// No more parsing after rule generation, we can stop the parser.
	if r.parser != nil {
		r.parser.Stop()
	}

}

// OnEnd implements GazellePhaseTransitionListener.
func (r *ScalaSourceCrossResolver) OnEnd() {
}

// IsLabelOwner implements the LabelOwner interface.
func (cr *ScalaSourceCrossResolver) IsLabelOwner(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	// if the label points to a rule that was generated by this extension
	if _, ok := ruleIndex(from); ok {
		return true
	}
	return false
}

// CrossResolve implements the CrossResolver interface.
func (r *ScalaSourceCrossResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) (result []resolve.FindResult) {
	if !(lang == r.lang || imp.Lang == r.lang) {
		return
	}

	if true {
		return
	}

	sym := imp.Imp

	// sc := getScalaConfig(c)
	// if providers, ok := r.providers[sym]; ok {
	// 	result = make([]resolve.FindResult, len(providers))
	// 	for i, p := range providers {
	// 		// log.Printf("source crossResolve %q provider hit %d: %v", imp.Imp, i, p.label)
	// 		result[i] = resolve.FindResult{Label: p.label}
	// 		if mapping, ok := sc.mapKindImportNames[p.rule.Kind]; ok {
	// 			result[i].Label = mapping.Rename(result[i].Label)
	// 		}
	// 	}
	// 	return
	// }

	sym = strings.TrimSuffix(sym, "._")

	if packages, ok := r.packages[sym]; ok {
		// pick the first result -- this might not be correct!
		result = make([]resolve.FindResult, len(packages))
		for i, p := range packages {
			// log.Printf("source crossResolve %q package hit %d: %v", imp.Imp, i, p.label)
			result[i] = resolve.FindResult{Label: p.label}
		}
		return
	}

	return
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
