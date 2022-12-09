package provider

import (
	"flag"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// ProvidedImports is the protoc.ImportProvider interface func.
type ProvidedImports func(lang, impLang string) map[label.Label][]string

// ProtobufProvider is a provider of symbols for the
// stackb/rules_proto gazelle extension.
type ProtobufProvider struct {
	lang                string
	impLang             string
	knownImportRegistry resolver.Scope
	importProvider      ProvidedImports
}

// NewProtobufProvider constructs a new provider.  The lang/impLang
// arguments are used to fetch the provided imports in the given importProvider
// struct.
func NewProtobufProvider(lang, impLang string, importProvider ProvidedImports) *ProtobufProvider {
	return &ProtobufProvider{
		lang:           lang,
		impLang:        impLang,
		importProvider: importProvider,
	}
}

// Name implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) Name() string {
	return "protobuf"
}

// RegisterFlags implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, registry resolver.Scope) error {
	p.knownImportRegistry = registry
	return nil
}

// OnResolve implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) OnResolve() {
	for from, symbols := range p.importProvider(p.lang, "package") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_PACKAGE, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "enum") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_OBJECT, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "message") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_CLASS, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "service") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_CLASS, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, p.impLang) {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_CLASS, symbol, from)
		}
	}
}

// CanProvide implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	return strings.HasSuffix(dep.Name, "proto_scala_library") ||
		strings.HasSuffix(dep.Name, "grpc_scala_library")
}

func (p *ProtobufProvider) putSymbol(impType sppb.ImportType, imp string, from label.Label) {
	p.knownImportRegistry.PutSymbol(resolver.NewSymbol(impType, imp, p.Name(), from))
}
