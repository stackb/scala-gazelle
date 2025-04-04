package provider

import (
	"flag"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// ProvidedImports is the protoc.ImportProvider interface func.  This is
// implemented by protoc.GlobalResolver from stackb/rules_proto.
type ProvidedImports func(lang, impLang string) map[label.Label][]string

// ProtobufProvider is a provider of symbols for the
// stackb/rules_proto gazelle extension.
type ProtobufProvider struct {
	lang           string
	impLang        string
	scope          resolver.Scope
	importProvider ProvidedImports
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
func (p *ProtobufProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, scope resolver.Scope) error {
	p.scope = scope

	return nil
}

// OnResolve implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) OnResolve() error {
	for from, symbols := range p.importProvider(p.lang, "package") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_PROTO_PACKAGE, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "enum") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_PROTO_ENUM, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "enumfield") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_PROTO_ENUM_FIELD, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "message") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_PROTO_MESSAGE, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "service") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_PROTO_SERVICE, symbol, from)
		}
	}
	for from, symbols := range p.importProvider(p.lang, "class") {
		for _, symbol := range symbols {
			p.putSymbol(sppb.ImportType_CLASS, symbol, from)
		}
	}
	return nil
}

// OnEnd implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) OnEnd() error {
	return nil
}

// CanProvide implements part of the resolver.SymbolProvider interface.
func (p *ProtobufProvider) CanProvide(dep *resolver.ImportLabel, expr build.Expr, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	return strings.HasSuffix(dep.Label.Name, "proto_scala_library") ||
		strings.HasSuffix(dep.Label.Name, "grpc_scala_library")
}

func (p *ProtobufProvider) putSymbol(impType sppb.ImportType, imp string, from label.Label) *resolver.Symbol {
	sym := resolver.NewSymbol(impType, imp, p.Name(), from)
	p.scope.PutSymbol(sym)
	return sym
}
