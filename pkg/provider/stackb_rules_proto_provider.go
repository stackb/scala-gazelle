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
func NewStackbRulesProtoProvider(lang, impLang string, importProvider protoc.ImportProvider, knownImportRegistry resolver.KnownImportRegistry) *StackbRulesProtoProvider {
	return &StackbRulesProtoProvider{
		lang:                lang,
		impLang:             impLang,
		importProvider:      importProvider,
		knownImportRegistry: knownImportRegistry,
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
	return nil
}

// OnResolve implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) OnResolve() {
	for from, symbols := range p.importProvider.Provided(p.lang, p.impLang) {
		for _, symbol := range symbols {
			p.knownImportRegistry.PutKnownImport(&resolver.KnownImport{
				Type:   sppb.ImportType_CLASS,
				Import: symbol,
				Label:  from,
			})
		}
	}
}

// CanProvide implements part of the resolver.KnownImportProvider interface.
func (p *StackbRulesProtoProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	return strings.HasSuffix(dep.Name, "proto_scala_library") ||
		strings.HasSuffix(dep.Name, "grpc_scala_library")
}