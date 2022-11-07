package crossresolve

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

type fromImportsProvider func(lang, impLang string) map[label.Label][]string

// NewProtoResolver creates a cross resolver for proto dependencies.
func NewProtoResolver(lang string, fromImportsProvider fromImportsProvider) *ProtoCrossResolver {
	return &ProtoCrossResolver{
		lang:                lang,
		fromImportsProvider: fromImportsProvider,
		importMap:           make(map[string]label.Label),
	}
}

// ProtoCrossResolver provides a cross-resolver for proto deps collected via protoc.GlobalResolver.
type ProtoCrossResolver struct {
	// lang is the language name to take provided proto rules for
	lang string
	// fromImportsProvider is typically protoc.GlobalResolver().Provided
	fromImportsProvider fromImportsProvider
	// importMap is a mapping from import string to the label that provides it
	importMap map[string]label.Label
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *ProtoCrossResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *ProtoCrossResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// OnResolve implements part of the GazellePhaseTransitionListener interface.
func (r *ProtoCrossResolver) OnResolve() {
	// gather proto imports
	for from, imports := range r.fromImportsProvider(r.lang, r.lang) {
		for _, imp := range imports {
			r.importMap[imp] = from
		}
	}
}

// OnEnd implements part of the GazellePhaseTransitionListener interface.
func (r *ProtoCrossResolver) OnEnd() {
}

// CrossResolve implements the CrossResolver interface.
func (r *ProtoCrossResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	if !crossResolverNameMatches(r.lang, lang, imp) {
		return nil
	}
	from, ok := r.importMap[imp.Imp]
	if !ok {
		return nil
	}
	return []resolve.FindResult{{Label: from}}
}
