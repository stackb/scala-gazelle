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

	"github.com/stackb/scala-gazelle/pkg/maven"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// RulesJvmExternalProvider is a provider of known imports for the
// bazelbuild/rules_jvm_external gazelle extension.
type RulesJvmExternalProvider struct {
	// the cross resolve language name to match
	lang string
	// raw comma-separated flag string
	pinnedMavenInstallFlagValue string
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
	fs.StringVar(&p.pinnedMavenInstallFlagValue,
		"pinned_maven_install_json_files",
		"",
		"comma-separated list of maven_install pinned deps files",
	)
}

// CheckFlags implements part of the resolver.KnownImportRegistry interface.
func (p *RulesJvmExternalProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, importRegistry resolver.KnownImportRegistry) error {
	if p.pinnedMavenInstallFlagValue == "" {
		return fmt.Errorf("maven cross resolver was requested but the -pinned_maven_install_json_files flag was not set")
	}

	filenames := strings.Split(p.pinnedMavenInstallFlagValue, ",")
	if len(filenames) == 0 {
		return fmt.Errorf("maven cross resolver was requested but the -pinned_maven_install_json_files flag did not specify any maven_install.json files")
	}

	p.resolvers = make([]maven.Resolver, len(filenames))

	for i, filename := range filenames {
		basename := filepath.Base(filename)
		if !strings.HasSuffix(basename, "_install.json") {
			return fmt.Errorf("maven cross resolver: -pinned_maven_install_json_files base name must match the pattern {name}_install.json (got %s)", basename)
		}
		name := basename[:len(basename)-len("_install.json")]
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(c.WorkDir, filename)
		}
		resolver, err := maven.NewResolver(filename, name, p.lang, func(format string, args ...interface{}) {
			log.Printf(format, args...)
		}, importRegistry.PutKnownImport)
		if err != nil {
			return fmt.Errorf("initializing maven resolver: %w", err)
		}
		p.resolvers[i] = resolver
	}

	return nil
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
