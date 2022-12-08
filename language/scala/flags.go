package scala

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const (
	scalaImportProviderFlagName   = "scala_import_provider"
	existingScalaRulesFlagName    = "existing_scala_rule"
	scalaGazelleCacheFileFlagName = "scala_gazelle_cache_file"
	cpuprofileFileFlagName        = "cpuprofile_file"
	memprofileFileFlagName        = "memprofile_file"
)

// RegisterFlags implements part of the language.Language interface
func (sl *scalaLang) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
	flags.StringVar(&sl.cacheFileFlagValue, scalaGazelleCacheFileFlagName, "", "optional path a cache file (.json or .pb)")
	flags.StringVar(&sl.cpuprofileFlagValue, cpuprofileFileFlagName, "", "optional path a cpuprofile file (.prof)")
	flags.StringVar(&sl.memprofileFlagValue, memprofileFileFlagName, "", "optional path a memory profile file (.prof)")
	flags.Var(&sl.importProviderNamesFlagValue, scalaImportProviderFlagName, "name of a known import provider implementation to enable")
	flags.Var(&sl.existingScalaRulesFlagValue, existingScalaRulesFlagName, "LOAD%NAME mapping for a custom existing_scala_rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scala_library'")

	sl.registerKnownImportProviders(flags, cmd, c)
}

func (sl *scalaLang) registerKnownImportProviders(flags *flag.FlagSet, cmd string, c *config.Config) {
	providers := resolver.GlobalKnownImportProviderRegistry().KnownImportProviders()
	for _, provider := range providers {
		provider.RegisterFlags(flags, cmd, c)
	}
}

// CheckFlags implements part of the language.Language interface
func (sl *scalaLang) CheckFlags(flags *flag.FlagSet, c *config.Config) error {
	sl.knownImportResolver = newKnownImportResolver(sl)

	if err := sl.setupKnownImportProviders(flags, c, sl.importProviderNamesFlagValue); err != nil {
		return err
	}
	if err := sl.setupExistingScalaRules(sl.existingScalaRulesFlagValue); err != nil {
		return err
	}
	if err := sl.setupCache(); err != nil {
		return err
	}
	if err := sl.setupCpuProfiling(c.WorkDir); err != nil {
		return err
	}
	if err := sl.setupMemoryProfiling(c.WorkDir); err != nil {
		return err
	}

	return nil
}

func (sl *scalaLang) setupKnownImportProviders(flags *flag.FlagSet, c *config.Config, names []string) error {
	providers, err := resolver.GetNamedKnownImportProviders(sl.importProviderNamesFlagValue)
	if err != nil {
		return err
	}
	for _, provider := range providers {
		if err := provider.CheckFlags(flags, c, sl); err != nil {
			return err
		}
	}
	sl.knownImportProviders = providers
	return nil
}

func (sl *scalaLang) setupExistingScalaRules(rules []string) error {
	for _, fqn := range rules {
		parts := strings.SplitN(fqn, "%", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid -existing_scala_rule flag value: wanted '%%' separated string, got %q", fqn)
		}
		if err := sl.setupExistingScalaRule(fqn, parts[0], parts[1]); err != nil {
			return err
		}
	}
	return nil
}

func (sl *scalaLang) setupExistingScalaRule(fqn, load, kind string) error {
	provider := &existingScalaRuleProvider{load, kind, isBinaryRule(kind)}
	return sl.ruleProviderRegistry.RegisterProvider(fqn, provider)
}

func (sl *scalaLang) setupCache() error {
	if sl.cacheFileFlagValue != "" {
		sl.cacheFileFlagValue = os.ExpandEnv(sl.cacheFileFlagValue)
		if err := sl.readCacheFile(); err != nil {
			// don't report error if the file does not exist yet
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("reading cache file: %w", err)
			}
		}
	}
	return nil
}

func (sl *scalaLang) setupCpuProfiling(workDir string) error {
	if sl.cpuprofileFlagValue != "" {
		if !filepath.IsAbs(sl.cpuprofileFlagValue) {
			sl.cpuprofileFlagValue = filepath.Join(workDir, sl.cpuprofileFlagValue)
		}
		f, err := os.Create(sl.cpuprofileFlagValue)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)

		log.Println("Collecting cpuprofile to", sl.cpuprofileFlagValue)
	}
	return nil
}

func (sl *scalaLang) setupMemoryProfiling(workDir string) error {
	if sl.memprofileFlagValue != "" {
		if !filepath.IsAbs(sl.memprofileFlagValue) {
			sl.memprofileFlagValue = filepath.Join(workDir, sl.memprofileFlagValue)
		}
	}
	return nil
}

func (sl *scalaLang) stopCpuProfiling() {
	if sl.cpuprofileFlagValue != "" {
		pprof.StopCPUProfile()
	}
}

func (sl *scalaLang) stopMemoryProfiling() {
	if sl.memprofileFlagValue != "" {
		f, err := os.Create(sl.memprofileFlagValue)
		if err != nil {
			log.Fatalf("creating memprofile: %v", err)
		}
		pprof.WriteHeapProfile(f)

		log.Println("Wrote memprofile to", sl.memprofileFlagValue)
	}
}

func isBinaryRule(kind string) bool {
	return strings.Contains(kind, "binary") || strings.Contains(kind, "test")
}
