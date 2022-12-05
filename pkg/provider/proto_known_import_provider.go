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

// StackbRulesProtoKnownImportProvider is a provider of known imports for the
// stackb/rules_proto gazelle extension.
type StackbRulesProtoKnownImportProvider struct {
	lang                string
	impLang             string
	knownImportRegistry resolver.KnownImportRegistry
	importProvider      protoc.ImportProvider
}

func NewStackbRulesProtoKnownImportProvider(lang, impLang string, importProvider protoc.ImportProvider, knownImportRegistry resolver.KnownImportRegistry) *StackbRulesProtoKnownImportProvider {
	return &StackbRulesProtoKnownImportProvider{
		lang:                lang,
		impLang:             impLang,
		importProvider:      importProvider,
		knownImportRegistry: knownImportRegistry,
	}
}

// Providers have canonical names
func (p *StackbRulesProtoKnownImportProvider) Name() string {
	return "github.com/stackb/rules_proto"
}

// RegisterFlags configures the flags.
func (p *StackbRulesProtoKnownImportProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags asserts that the flags are correct.
func (p *StackbRulesProtoKnownImportProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, registry resolver.KnownImportRegistry) error {
	return nil
}

// Providers typically manage a particular sub-space of labels.  For example,
// the maven resolver may return true for labels like
// "@maven//:junit_junit". The rule Index can be used to consult what type
// of label from is, based on the rule characteristics.
func (p *StackbRulesProtoKnownImportProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	return strings.HasSuffix(dep.Name, "proto_scala_library") ||
		strings.HasSuffix(dep.Name, "grpc_scala_library")
}

// OnResolve is a lifecycle hook that gets called when the resolve phase is
// beginning.
func (p *StackbRulesProtoKnownImportProvider) OnResolve() {
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
