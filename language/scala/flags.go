package scala

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const (
	scalaImportProviderFlagName   = "scala_import_provider"
	scalaExistingRulesFlagName    = "scala_existing_rule"
	scalaGazelleCacheFileFlagName = "scala_gazelle_cache_file"
)

// RegisterFlags implements part of the language.Language interface
func (sl *scalaLang) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
	getOrCreateScalaConfig(c, "" /* rel="" */, sl) // ignoring return value, only want side-effect

	flags.StringVar(&sl.cacheFileFlagValue, scalaGazelleCacheFileFlagName, "", "optional path the a cache file (.json or .pb)")
	flags.Var(&sl.importProviderNamesFlagValue, scalaImportProviderFlagName, "name of a known import provider implementation to enable")
	flags.Var(&sl.scalaExistingRulesFlagValue, scalaExistingRulesFlagName, "LOAD%NAME mapping for a custom scala_existing_rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scala_library'")

	for _, provider := range sl.knownImportProviders {
		provider.RegisterFlags(flags, cmd, c)
	}
}

// CheckFlags implements part of the language.Language interface
func (sl *scalaLang) CheckFlags(flags *flag.FlagSet, c *config.Config) error {
	// initialize the resolver implementation
	sl.knownImportResolver = NewKnownImportResolver(sl)

	if sl.cacheFileFlagValue != "" {
		sl.cacheFileFlagValue = os.ExpandEnv(sl.cacheFileFlagValue)
		if err := sl.readCacheFile(); err != nil {
			// don't report error if the file does not exist yet
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("reading cache file: %w", err)
			}
		}
	}

	if err := parseScalaExistingRules(sl.scalaExistingRulesFlagValue); err != nil {
		return err
	}

	sl.knownImportProviders = filterNamedKnownImportProviders(
		sl.knownImportProviders, sl.importProviderNamesFlagValue)
	for _, provider := range sl.knownImportProviders {
		provider.CheckFlags(flags, c, sl)
	}

	return nil
}

func filterNamedKnownImportProviders(current []resolver.KnownImportProvider, names []string) (want []resolver.KnownImportProvider) {
	for _, name := range names {
		for _, provider := range current {
			if name == provider.Name() {
				want = append(want, provider)
			}
		}
	}
	return
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
