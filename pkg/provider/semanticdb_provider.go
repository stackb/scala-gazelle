package provider

import (
	"flag"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/semanticdb"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

// NewSemanticdbProvider constructs a new NewSemanticdbProvider.
func NewSemanticdbProvider(delegate parser.Parser) *SemanticdbProvider {
	infoMap := new(spb.InfoMap)
	infoMap.Entries = make(map[string]string)

	return &SemanticdbProvider{
		delegate: delegate,
		jarFiles: make(collections.StringSlice, 0),
		docs:     make(map[string]*spb.TextDocument),
		infoMap:  infoMap,
	}
}

// SemanticdbProvider is provider for scala source files. If -scala_source_index_in
// is configured, the given source index will be used to bootstrap the internal
// cache.  At runtime the .ParseScalaRule function can be used to parse scala
// files.  If the cache already has an entry for the filename with matching
// sha256, the cache hit will be used.
type SemanticdbProvider struct {
	// delegate is an instance of the inner parent parser
	delegate parser.Parser
	// indexFile is the name of the file that should be parsed as a InfoMap
	indexFile string
	// jarFiles is a repeatable list of jars to include in the index
	jarFiles collections.StringSlice
	// configWorkDir is the Config.WorkDir that gives us the abs path prefix for
	// filenames.
	configWorkDir string

	// docs is a map of known text documents
	docs map[string]*spb.TextDocument
	// infoMap is the parsed infoMap proto (from index file)
	infoMap *spb.InfoMap
}

// Name implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) Name() string {
	return "semanticdb"
}

// RegisterFlags implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
	flags.StringVar(&r.indexFile, "semanticdb_index_file", "", "path to the semanticdb index file")
	flags.Var(&r.jarFiles, "semanticdb_jar_file", "path to a scala jar that contains semanticdb meta-inf")
}

// CheckFlags implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) CheckFlags(flags *flag.FlagSet, c *config.Config, scope resolver.Scope) error {
	r.configWorkDir = c.WorkDir

	if r.indexFile != "" {
		if err := r.parseIndex(r.indexFile); err != nil {
			return err
		}
	}
	for _, jarFile := range r.jarFiles {
		if err := r.parseJarFile(jarFile); err != nil {
			return err
		}
	}
	return nil
}

// OnResolve implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) OnResolve() error {
	return nil
}

// OnEnd implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) OnEnd() error {
	return nil
}

// CanProvide implements the resolver.SymbolProvider interface.
func (cr *SemanticdbProvider) CanProvide(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	return false
}

// ParseScalaRule implements scalarule.Parser
func (r *SemanticdbProvider) ParseScalaRule(kind string, from label.Label, dir string, srcs ...string) (*sppb.Rule, error) {
	rule, err := r.delegate.ParseScalaRule(kind, from, dir, srcs...)
	if err != nil {
		return nil, err
	}
	for _, file := range rule.Files {
		r.visitFile(from.Pkg, file)
	}
	return rule, nil
}

// LoadScalaRule loads the given state.
func (r *SemanticdbProvider) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	return r.delegate.LoadScalaRule(from, rule)
}

func (r *SemanticdbProvider) loadSemanticInfoFile(wantUri, path string) (got *spb.TextDocument, err error) {
	var pb spb.TextDocuments
	if err = protobuf.ReadFile(path, &pb); err != nil {
		return nil, fmt.Errorf("attempted read of semanticdb info file %s: %v", path, err)
	}
	for _, d := range pb.Documents {
		r.docs[d.Uri] = d
		if d.Uri == wantUri {
			got = d
		}
	}
	return
}

func (r *SemanticdbProvider) visitFile(pkg string, file *sppb.File) (*sppb.File, error) {
	uri := path.Join(pkg, file.Filename)

	// do we have a parsed doc already?
	if doc, ok := r.docs[uri]; ok {
		return mergeImports(doc, file)
	}

	// can we locate the info file?
	path, exists := r.infoMap.Entries[uri]
	if !exists {
		return nil, fmt.Errorf("no semantic info available for: %s", uri)
	}

	// load and merge it
	abspath := filepath.Join(r.configWorkDir, path)
	if doc, err := r.loadSemanticInfoFile(uri, abspath); err != nil {
		return nil, err
	} else {
		return mergeImports(doc, file)
	}
}

func (r *SemanticdbProvider) parseIndex(_ string) error {
	if err := protobuf.ReadFile(r.indexFile, r.infoMap); err != nil {
		return err
	}
	return nil
}

func (r *SemanticdbProvider) parseJarFile(filename string) error {
	collections.ListFiles("..")

	list, err := semanticdb.ReadJarFile(filename)
	if err != nil {
		return err
	}
	for _, docs := range list {
		for _, doc := range docs.Documents {
			if err := r.addTextDocument(doc); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *SemanticdbProvider) addTextDocument(doc *spb.TextDocument) error {
	if _, exists := r.docs[doc.Uri]; exists {
		return fmt.Errorf("text doc already registered: %s", doc.Uri)
	}
	r.docs[doc.Uri] = doc
	return nil
}

func mergeImports(doc *spb.TextDocument, file *sppb.File) (*sppb.File, error) {
	next, err := semanticdb.ToFile(doc)
	if err != nil {
		return nil, fmt.Errorf("error while gathering semantic info for file: %v", err)
	}
	original := deduplicateAndSort(file.Imports)
	imports := deduplicateAndSort(append(file.Imports, next.Imports...))

	if diff := cmp.Diff(original, imports); diff != "" {
		log.Println(doc.Uri, "imports diff:\n", diff)
	}

	file.Imports = imports

	return file, nil
}

// deduplicateAndSort removes duplicate entries and sorts the list
func deduplicateAndSort(in []string) (out []string) {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]bool)
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return
}
