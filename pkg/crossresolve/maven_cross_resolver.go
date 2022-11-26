package crossresolve

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/maven"
)

// NewMavenResolver creates a cross resolver for maven dependencies.
// the "lang" argument specifies the .CrossResolve lang name to match against
// (typically "scala").
func NewMavenResolver(lang string) *MavenCrossResolver {
	return &MavenCrossResolver{lang: lang}
}

// MavenCrossResolver provides a cross-resolver for maven deps.
type MavenCrossResolver struct {
	// the cross resolve language name to match
	lang string
	// raw comma-separated flag string
	pinnedMavenInstallFlagValue string
	// internal resolver instances, preserving order of the flag
	resolvers []maven.Resolver
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (cr *MavenCrossResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&cr.pinnedMavenInstallFlagValue,
		"pinned_maven_install_json_files",
		"",
		"comma-separated list of maven_install pinned deps files",
	)
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (cr *MavenCrossResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if cr.pinnedMavenInstallFlagValue == "" {
		return fmt.Errorf("maven cross resolver was requested but the -pinned_maven_install_json_files flag was not set")
	}

	filenames := strings.Split(cr.pinnedMavenInstallFlagValue, ",")
	if len(filenames) == 0 {
		return fmt.Errorf("maven cross resolver was requested but the -pinned_maven_install_json_files flag did not specify any maven_install.json files")
	}

	cr.resolvers = make([]maven.Resolver, len(filenames))

	for i, filename := range filenames {
		basename := filepath.Base(filename)
		if !strings.HasSuffix(basename, "_install.json") {
			return fmt.Errorf("maven cross resolver: -pinned_maven_install_json_files base name must match the pattern {name}_install.json (got %s)", basename)
		}
		name := basename[:len(basename)-len("_install.json")]
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(c.WorkDir, filename)
		}
		resolver, err := maven.NewResolver(filename, name)
		if err != nil {
			return fmt.Errorf("initializing maven resolver: %w", err)
		}
		cr.resolvers[i] = resolver
	}

	return nil
}

// IsOwner implements the LabelOwner interface.
func (cr *MavenCrossResolver) IsOwner(from label.Label, ruleIndex func(from label.Label) (*rule.Rule, bool)) bool {
	// if the resolver is nil, checkflags was never called and we can infer that
	// this resolver is not enabled
	if len(cr.resolvers) == 0 {
		return false
	}

	// find the first resolver that manages the given workspace
	for _, resolver := range cr.resolvers {
		if from.Repo == resolver.Name() {
			return true
		}
	}

	return false
}

// CrossResolve implements the CrossResolver interface.
func (cr *MavenCrossResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) (result []resolve.FindResult) {
	if len(cr.resolvers) == 0 {
		return
	}
	if !crossResolverNameMatches(cr.lang, lang, imp) {
		return
	}
	for _, resolver := range cr.resolvers {
		if from, _, err := resolver.ResolveWithDirectDependencies(imp.Imp); err == nil {
			// return []resolve.FindResult{{Label: from, Embeds: embeds}}
			return []resolve.FindResult{{Label: from}}
		}
	}
	return
}
