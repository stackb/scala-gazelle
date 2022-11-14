package scala

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

// RegisterFlags implements part of the language.Language interface
func (sl *scalaLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	getOrCreateScalaConfig(c) // ignoring return value, only want side-effect

	fs.IntVar(&sl.totalPackageCount, "total_package_count", 0, "number of total packages for the workspace (used for progress estimation)")
	fs.StringVar(&sl.resolverNames, "scala_resolvers", "maven,proto", "comma-separated list of scala cross-resolver implementations to enable")

	// all known cross-resolvers can register flags, but do it in repeatable order
	resolvers := crossresolve.Resolvers().ByName()
	names := make([]string, 0, len(resolvers))
	for name := range resolvers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		resolvers[name].RegisterFlags(fs, cmd, c)
	}

	sl.scalaCompiler.RegisterFlags(fs, cmd, c)
}

// CheckFlags implements part of the language.Language interface
func (sl *scalaLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	for _, name := range strings.Split(sl.resolverNames, ",") {
		resolver, err := crossresolve.Resolvers().LookupResolver(name)
		if err != nil {
			return fmt.Errorf("-scala_resolver %s error: %v", name, err)
		}
		if err := resolver.CheckFlags(fs, c); err != nil {
			return fmt.Errorf("check flags %s: %w", name, err)
		}
		sl.resolvers[name] = resolver
	}

	if err := sl.scalaCompiler.CheckFlags(fs, c); err != nil {
		return err
	}

	return nil
}
