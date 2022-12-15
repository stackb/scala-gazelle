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
	scalaSymbolProviderFlagName   = "scala_symbol_provider"
	scalaConflictResolverFlagName = "scala_conflict_resolver"
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
	flags.Var(&sl.symbolProviderNamesFlagValue, scalaSymbolProviderFlagName, "name of a symbol provider implementation to enable")
	flags.Var(&sl.conflictResolverNamesFlagValue, scalaConflictResolverFlagName, "name of a conflict resolver implementation to enable")
	flags.Var(&sl.existingScalaRulesFlagValue, existingScalaRulesFlagName, "LOAD%NAME mapping for a custom existing_scala_rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scala_library'")

	sl.registerSymbolProviders(flags, cmd, c)
	sl.registerConflictResolvers(flags, cmd, c)
}

func (sl *scalaLang) registerSymbolProviders(flags *flag.FlagSet, cmd string, c *config.Config) {
	providers := resolver.GlobalSymbolProviderRegistry().SymbolProviders()
	for _, provider := range providers {
		provider.RegisterFlags(flags, cmd, c)
	}
}

func (sl *scalaLang) registerConflictResolvers(flags *flag.FlagSet, cmd string, c *config.Config) {
	resolver := resolver.GlobalConflictResolvers()
	for _, provider := range resolver {
		provider.RegisterFlags(flags, cmd, c)
	}
}

// CheckFlags implements part of the language.Language interface
func (sl *scalaLang) CheckFlags(flags *flag.FlagSet, c *config.Config) error {
	sl.symbolResolver = newUniverseResolver(sl)

	if err := sl.setupSymbolProviders(flags, c, sl.symbolProviderNamesFlagValue); err != nil {
		return err
	}
	if err := sl.setupConflictResolvers(flags, c, sl.conflictResolverNamesFlagValue); err != nil {
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

func (sl *scalaLang) setupSymbolProviders(flags *flag.FlagSet, c *config.Config, names []string) error {
	providers, err := resolver.GetNamedSymbolProviders(sl.symbolProviderNamesFlagValue)
	if err != nil {
		return err
	}
	for _, provider := range providers {
		if err := provider.CheckFlags(flags, c, sl); err != nil {
			return err
		}
	}
	sl.symbolProviders = providers
	return nil
}

func (sl *scalaLang) setupConflictResolvers(flags *flag.FlagSet, c *config.Config, names []string) error {
	for _, name := range sl.conflictResolverNamesFlagValue {
		resolver, ok := resolver.GlobalConflictResolverRegistry().GetConflictResolver(name)
		if !ok {
			return fmt.Errorf("-%s not found: %q", scalaConflictResolverFlagName, name)
		}
		if err := resolver.CheckFlags(flags, c); err != nil {
			return err
		}
		sl.conflictResolvers[name] = resolver
	}
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
	provider := &existingScalaRuleProvider{load, kind}
	return sl.ruleProviderRegistry.RegisterProvider(fqn, provider)
}

func (sl *scalaLang) setupCache() error {
	if sl.cacheFileFlagValue != "" {
		sl.cacheFileFlagValue = os.ExpandEnv(sl.cacheFileFlagValue)
		if err := sl.readScalaRuleCacheFile(); err != nil {
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
