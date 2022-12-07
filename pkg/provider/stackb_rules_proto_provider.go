package provider

import (
	"flag"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/rules_proto/pkg/protoc"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// StackbRulesProtoProvider is a provider of known imports for the
// stackb/rules_proto gazelle extension.
type StackbRulesProtoProvider struct {
	lang                string
	impLang             string
	knownImportRegistry resolver.KnownImportRegistry
	importProvider      protoc.ImportProvider
}

// NewStackbRulesProtoProvider constructs a new provider.  The lang/impLang
// arguments are used to fetch the provided imports in the given importProvider
// struct.
func NewStackbRulesProtoProvider(lang, impLang string, importProvider protoc.ImportProvider) *StackbRulesProtoProvider {
	return &StackbRulesProtoProvider{
		lang:           lang,
		impLang:        impLang,
		importProvider: importProvider,
	}
}

// Name implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) Name() string {
	return "github.com/stackb/rules_proto"
}

// RegisterFlags implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, registry resolver.KnownImportRegistry) error {
	p.knownImportRegistry = registry
	return nil
}

// OnResolve implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) OnResolve() {
	for from, symbols := range p.importProvider.Provided(p.lang, "package") {
		for _, symbol := range symbols {
			p.putKnownImport(sppb.ImportType_PACKAGE, symbol, from)
		}
	}
	for from, symbols := range p.importProvider.Provided(p.lang, "enum") {
		for _, symbol := range symbols {
			p.putKnownImport(sppb.ImportType_OBJECT, symbol, from)
		}
	}
	for from, symbols := range p.importProvider.Provided(p.lang, "message") {
		for _, symbol := range symbols {
			p.putKnownImport(sppb.ImportType_CLASS, symbol, from)
		}
	}
	for from, symbols := range p.importProvider.Provided(p.lang, "service") {
		for _, symbol := range symbols {
			p.putKnownImport(sppb.ImportType_CLASS, symbol, from)
		}
	}
	for from, symbols := range p.importProvider.Provided(p.lang, p.impLang) {
		for _, symbol := range symbols {
			p.putKnownImport(sppb.ImportType_CLASS, symbol, from)
		}
	}
}

// CanProvide implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	return strings.HasSuffix(dep.Name, "proto_scala_library") ||
		strings.HasSuffix(dep.Name, "grpc_scala_library")
}

func (p *StackbRulesProtoProvider) putKnownImport(impType sppb.ImportType, imp string, from label.Label) {
	p.knownImportRegistry.PutKnownImport(resolver.NewKnownImport(impType, imp, p.Name(), from))
}
