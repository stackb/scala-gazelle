package scala

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

const (
	totalPackageCountFlagName  = "total_package_count"
	scalaResolversFlagName     = "scala_resolvers"
	scalaExistingRulesFlagName = "scala_existing_rule"
)

// RegisterFlags implements part of the language.Language interface
func (sl *scalaLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	getOrCreateScalaConfig(sl, c, "" /* rel="" */) // ignoring return value, only want side-effect

	fs.IntVar(&sl.totalPackageCount, totalPackageCountFlagName, 0, "number of total packages for the workspace (used for progress estimation)")
	fs.StringVar(&sl.resolverNames, scalaResolversFlagName, "maven,proto,source", "comma-separated list of scala cross-resolver implementations to enable")
	fs.Var(&sl.scalaExistingRules, scalaExistingRulesFlagName, "LOAD%NAME mapping for a custom scala_existing_rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scala_library'")

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
	if err := parseScalaExistingRules(sl.scalaExistingRules); err != nil {
		return err
	}
	for _, name := range strings.Split(sl.resolverNames, ",") {
		resolver, err := crossresolve.Resolvers().LookupResolver(name)
		if err != nil {
			return fmt.Errorf("-scala_resolvers %q error: %v", name, err)
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

func parseScalaExistingRules(rules []string) error {
	for _, fqn := range rules {
		parts := strings.SplitN(fqn, "%", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid -scala_existing_rule flag value: wanted '%%' separated string, got %q", fqn)
		}
		load := parts[0]
		kind := parts[1]
		isBinaryRule := strings.Contains(kind, "binary") || strings.Contains(kind, "test")
		Rules().MustRegisterRule(fqn, &scalaExistingRule{load, kind, isBinaryRule})
	}
	return nil
}

type stringSliceFlags []string

func (i *stringSliceFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *stringSliceFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
