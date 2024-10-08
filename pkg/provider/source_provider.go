package provider

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

type progressFunc func(msg string)

// NewSourceProvider constructs a new NewSourceProvider.
func NewSourceProvider(progress progressFunc) *SourceProvider {
	return &SourceProvider{
		progress: progress,
		parser:   parser.NewScalametaParser(),
	}
}

// SourceProvider is provider for scala source files. If -scala_source_index_in
// is configured, the given source index will be used to bootstrap the internal
// cache.  At runtime the .ParseScalaRule function can be used to parse scala
// files.  If the cache already has an entry for the filename with matching
// sha256, the cache hit will be used.
type SourceProvider struct {
	progress progressFunc
	// scope is the target we provide symbols to
	scope resolver.Scope
	// parser is an instance of the scala source parser
	parser *parser.ScalametaParser
}

// Name implements part of the resolver.SymbolProvider interface.
func (r *SourceProvider) Name() string {
	return "source"
}

// RegisterFlags implements part of the resolver.SymbolProvider interface.
func (r *SourceProvider) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.SymbolProvider interface.
func (r *SourceProvider) CheckFlags(flags *flag.FlagSet, c *config.Config, scope resolver.Scope) error {
	r.scope = scope
	return r.start()
}

// OnResolve implements part of the resolver.SymbolProvider interface.
func (r *SourceProvider) OnResolve() error {
	r.parser.Stop()
	return nil
}

// OnEnd implements part of the resolver.SymbolProvider interface.
func (r *SourceProvider) OnEnd() error {
	return nil
}

// CanProvide implements the resolver.SymbolProvider interface.
func (cr *SourceProvider) CanProvide(dep *resolver.ImportLabel, expr build.Expr, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	// if the label points to a rule that was generated by this extension
	if _, ok := ruleIndex(dep.Label); ok {
		return true
	}
	return false
}

// start begins the parser process.
func (r *SourceProvider) start() error {
	if err := r.parser.Start(); err != nil {
		return fmt.Errorf("starting parser: %w", err)
	}
	return nil
}

// ParseScalaRule implements scalarule.Parser
func (r *SourceProvider) ParseScalaRule(kind string, from label.Label, dir string, srcs ...string) (*sppb.Rule, error) {
	if len(srcs) == 0 {
		return nil, nil
	}
	sort.Strings(srcs)

	t1 := time.Now()

	files, err := r.parseFiles(dir, srcs)
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		a := files[i]
		b := files[j]
		return a.Filename < b.Filename
	})

	for _, file := range files {
		if err := r.loadScalaFile(from, kind, file); err != nil {
			return nil, err
		}
	}

	t2 := time.Since(t1).Round(1 * time.Millisecond)
	if true {
		log.Printf("Parsed %s%%%s (%d files, %v)", from, kind, len(files), t2)
	}

	return &sppb.Rule{
		Label:           from.String(),
		Kind:            kind,
		Files:           files,
		ParseTimeMillis: t2.Milliseconds(),
	}, nil
}

func (r *SourceProvider) parseFiles(dir string, srcs []string) ([]*sppb.File, error) {
	filenames := make([]string, len(srcs))
	for i, src := range srcs {
		filenames[i] = filepath.Join(dir, src)
	}

	response, err := r.parser.Parse(context.Background(), &sppb.ParseRequest{
		Filenames: filenames,
	})
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("parser error: %s", response.Error)
	}

	// check for errors and remove dir prefixes
	for _, file := range response.Files {
		if file.Error != "" {
			return nil, fmt.Errorf("%s parse error: %s", file.Filename, file.Error)
		}
		file.Filename = strings.TrimPrefix(strings.TrimPrefix(file.Filename, dir), "/")
	}

	return response.Files, nil
}

// LoadScalaRule loads the given state.
func (r *SourceProvider) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	for _, file := range rule.Files {
		if err := r.loadScalaFile(from, rule.Kind, file); err != nil {
			return err
		}
	}
	return nil
}

func (r *SourceProvider) loadScalaFile(from label.Label, kind string, file *sppb.File) error {
	for _, imp := range file.Classes {
		r.putSymbol(from, kind, imp, sppb.ImportType_CLASS)
	}
	for _, imp := range file.Objects {
		r.putSymbol(from, kind, imp, sppb.ImportType_OBJECT)
	}
	for _, imp := range file.Traits {
		r.putSymbol(from, kind, imp, sppb.ImportType_TRAIT)
	}
	for _, imp := range file.Types {
		r.putSymbol(from, kind, imp, sppb.ImportType_TYPE)
	}
	for _, imp := range file.Vals {
		r.putSymbol(from, kind, imp, sppb.ImportType_VALUE)
	}
	return nil
}

func (r *SourceProvider) putSymbol(from label.Label, kind, imp string, impType sppb.ImportType) {
	r.scope.PutSymbol(resolver.NewSymbol(impType, imp, kind, from))
}
