package provider

import (
	"flag"
	"fmt"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/semanticdb"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

const semanticDbName = "semanticdb"

// NewSemanticdbProvider constructs a new NewSemanticdbProvider.
func NewSemanticdbProvider(delegate parser.Parser) *SemanticdbProvider {
	return &SemanticdbProvider{
		delegate: delegate,
		jarFiles: make(collections.StringSlice, 0),
		docs:     make(map[string]*spb.TextDocument),
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
	// docs is a map of known text documents
	docs map[string]*spb.TextDocument
}

// Name implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) Name() string {
	return semanticDbName
}

// RegisterFlags implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
	flags.StringVar(&r.indexFile,
		"semanticdb_index_file",
		"",
		"path to the semanticdb index file")
	flags.Var(&r.jarFiles,
		"semanticdb_jar_file",
		"path to a scala jar that contains semanticdb meta-inf")
}

// CheckFlags implements part of the resolver.SymbolProvider interface.
func (r *SemanticdbProvider) CheckFlags(flags *flag.FlagSet, c *config.Config, scope resolver.Scope) error {

	semanticdb.SetGlobalScope(scope)

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
func (cr *SemanticdbProvider) CanProvide(dep *resolver.ImportLabel, expr build.Expr, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	hasSemanticDbSuffixComment := false
	for _, c := range append(expr.Comment().Before, expr.Comment().Suffix...) {
		text := strings.TrimSpace(strings.TrimPrefix(c.Token, "#"))
		if text == semanticDbName {
			hasSemanticDbSuffixComment = true
			break
		}
	}
	if !hasSemanticDbSuffixComment {
		return false
	}

	// at this point in the function, we know the dep was labeled as a semanticdb-managed dep.  If the resolved import
	if dep.Import.Source == nil {
		return false
	}

	// if the source file for the import has semanticdb info, we expect it to
	// successfully resolve again, so return true.  If the Source file was not
	// augmented with semanticdb info, we assume that the '# semanticdb' comment
	// originated from a previous gazelle run.
	return len(dep.Import.Source.SemanticImports) > 0
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

func (r *SemanticdbProvider) visitFile(pkg string, file *sppb.File) error {
	uri := path.Join(pkg, file.Filename)
	if doc, ok := r.docs[uri]; ok {
		file.SemanticImports = semanticdb.SemanticImports(doc)
	}
	return nil
}

func (r *SemanticdbProvider) parseIndex(_ string) error {
	var docs spb.TextDocuments
	if err := protobuf.ReadFile(r.indexFile, &docs); err != nil {
		return err
	}
	for _, doc := range docs.Documents {
		r.docs[doc.Uri] = doc
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
