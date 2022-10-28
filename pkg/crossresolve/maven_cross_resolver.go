package crossresolve

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

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
	// absolute or relative path to the maven_install.json file
	mavenInstallFile string
	// name of the workspace that corresponds to the maven_install.json file
	mavenWorkspaceName string
	// internal resolver instance
	resolver maven.Resolver
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *MavenCrossResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.mavenInstallFile,
		"maven_install_file",
		"",
		"pinned maven_install.json deps",
	)
	fs.StringVar(&r.mavenWorkspaceName,
		"maven_workspace_name",
		"maven",
		"name of the maven external workspace",
	)
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *MavenCrossResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.mavenInstallFile != "" {
		// assume relative to workspace
		mavenInstallFile := filepath.Join(c.WorkDir, r.mavenInstallFile)
		resolver, err := maven.NewResolver(mavenInstallFile, r.mavenWorkspaceName)
		if err != nil {
			return fmt.Errorf("initializing maven resolver: %w", err)
		}
		r.resolver = resolver
	} else {
		return fmt.Errorf("maven cross resolver (-maven_install_file flag not set)")
	}
	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *MavenCrossResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) (result []resolve.FindResult) {
	if r.resolver == nil {
		return
	}
	if !crossResolverNameMatches(r.lang, lang, imp) {
		return
	}
	match, err := r.resolver.Resolve(imp.Imp)
	if err != nil {
		return
	}
	if match == label.NoLabel {
		return
	}
	return []resolve.FindResult{{Label: match}}
}
