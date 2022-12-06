package provider

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/maven"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// RulesJvmExternalProvider is a provider of known imports for the
// bazelbuild/rules_jvm_external gazelle extension.
type RulesJvmExternalProvider struct {
	// the cross resolve language name to match
	lang string
	// raw comma-separated flag string
	mavenInstallJSONFiles collections.StringSlice
	// internal resolver instances, preserving order of the flag
	resolvers []maven.Resolver
}

// NewRulesJvmExternalProvider constructs a new provider having the
// given resolving lang/impLang as well as the importRegistry instance.
func NewRulesJvmExternalProvider(lang string) *RulesJvmExternalProvider {
	return &RulesJvmExternalProvider{
		lang: lang,
	}
}

// Name implements part of the resolver.KnownImportRegistry interface.
func (p *RulesJvmExternalProvider) Name() string {
	return "github.com/bazelbuild/rules_jvm_external"
}

// RegisterFlags implements part of the resolver.KnownImportRegistry interface.
func (p *RulesJvmExternalProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.Var(&p.mavenInstallJSONFiles, "maven_install_json_file", "path to maven_install.json file")
}

// CheckFlags implements part of the resolver.KnownImportRegistry interface.
func (p *RulesJvmExternalProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, importRegistry resolver.KnownImportRegistry) error {
	p.resolvers = make([]maven.Resolver, len(p.mavenInstallJSONFiles))

	for i, filename := range p.mavenInstallJSONFiles {
		resolver, err := p.loadFile(c.WorkDir, filename, importRegistry)
		if err != nil {
			return err
		}
		p.resolvers[i] = resolver
	}

	return nil
}

func (p *RulesJvmExternalProvider) loadFile(dir string, filename string, importRegistry resolver.KnownImportRegistry) (maven.Resolver, error) {
	basename := filepath.Base(filename)
	if !strings.HasSuffix(basename, "_install.json") {
		return nil, fmt.Errorf("maven cross resolver: -maven_install_json_file base name must match the pattern {name}_install.json (got %s)", basename)
	}
	name := basename[:len(basename)-len("_install.json")]
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(dir, filename)
	}
	resolver, err := maven.NewResolver(filename, name, p.lang, func(format string, args ...interface{}) {
		log.Printf(format, args...)
	}, importRegistry.PutKnownImport)
	if err != nil {
		return nil, fmt.Errorf("initializing maven resolver: %w", err)
	}
	return resolver, nil
}

// CanProvide implements part of the resolver.KnownImportRegistry interface.
func (p *RulesJvmExternalProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	// if the resolver is nil, checkflags was never called and we can infer that
	// this resolver is not enabled
	if len(p.resolvers) == 0 {
		return false
	}

	// find the first resolver that manages the given workspace
	for _, resolver := range p.resolvers {
		if dep.Repo == resolver.Name() {
			return true
		}
	}

	return false
}

// OnResolve implements part of the resolver.KnownImportRegistry interface.
func (p *RulesJvmExternalProvider) OnResolve() {
}
