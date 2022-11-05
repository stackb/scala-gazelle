package scala

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/maven"
)

func newMavenResolver() *mavenResolver {
	return &mavenResolver{}
}

// mavenResolver provides a cross-resolver for maven deps.
type mavenResolver struct {
	resolver           maven.Resolver
	mavenInstallFile   string
	mavenWorkspaceName string
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *mavenResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.mavenInstallFile, "maven_install_file", "", "pinned maven_install.json deps")
	fs.StringVar(&r.mavenWorkspaceName, "maven_workspace_name", "maven", "name of the maven external workspace")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *mavenResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.mavenInstallFile != "" {
		// assume relative to workspace
		mavenInstallFile := filepath.Join(c.WorkDir, r.mavenInstallFile)
		resolver, err := maven.NewResolver(mavenInstallFile, r.mavenWorkspaceName)
		if err != nil {
			return fmt.Errorf("initializing maven resolver: %v", err)
		}
		r.resolver = resolver
	} else {
		log.Println("skipping maven resolution (-maven_install_file flag not set)")
	}
	return nil
}

// CrossResolve implements the CrossResolver interface.
func (r *mavenResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) (result []resolve.FindResult) {
	if r.resolver == nil {
		return
	}
	if !(lang == ScalaLangName || imp.Lang == ScalaLangName) {
		return
	}
	log.Println("maven resolver: resolving", imp.Imp)
	match, err := r.resolver.Resolve(imp.Imp)
	if err != nil {
		return
	}
	if match == label.NoLabel {
		return
	}
	return []resolve.FindResult{{Label: match}}
}
