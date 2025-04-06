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

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const (
	scalaSymbolProviderFlagName          = "scala_symbol_provider"
	scalaConflictResolverFlagName        = "scala_conflict_resolver"
	scalaDepsCleanerFlagName             = "scala_deps_cleaner"
	existingScalaBinaryRuleFlagName      = "existing_scala_binary_rule"
	existingScalaLibraryRuleFlagName     = "existing_scala_library_rule"
	existingScalaTestRuleFlagName        = "existing_scala_test_rule"
	existingScalaRuleCoverageFlagName    = "existing_scala_rule_coverage"
	scalaGazelleCacheFileFlagName        = "scala_gazelle_cache_file"
	scalaGazelleImportsFileFlagName      = "scala_gazelle_imports_file"
	scalaGazelleDebugProcessFileFlagName = "scala_gazelle_debug_process"
	scalaGazelleCacheKeyFlagName         = "scala_gazelle_cache_key"
	scalaGazellePrintCacheKeyFlagName    = "scala_gazelle_print_cache_key"
	cpuprofileFileFlagName               = "cpuprofile_file"
	memprofileFileFlagName               = "memprofile_file"
	logFileFlagName                      = "log_file"
)

// RegisterFlags implements part of the language.Language interface
func (sl *scalaLang) RegisterFlags(flags *flag.FlagSet, cmd string, c *config.Config) {
	sl.phaseTransition("config")

	flags.BoolVar(&sl.debugProcessFlagValue, scalaGazelleDebugProcessFileFlagName, false, "if true, prints the process ID and waits for debugger to attach")
	flags.BoolVar(&sl.printCacheKey, scalaGazellePrintCacheKeyFlagName, true, "if a cache key is set, print the version for auditing purposes")
	flags.BoolVar(&sl.existingScalaRuleCoverageFlagValue, existingScalaRuleCoverageFlagName, true, "report coverage statistics")
	flags.StringVar(&sl.cacheFileFlagValue, scalaGazelleCacheFileFlagName, "", "optional path a cache file (.json or .pb)")
	flags.StringVar(&sl.importsFileFlagValue, scalaGazelleImportsFileFlagName, "", "optional path to an imports file where resolved imports should be written (.json or .pb)")
	flags.StringVar(&sl.cacheKeyFlagValue, scalaGazelleCacheKeyFlagName, "", "optional string that can be used to bust the cache file")
	flags.StringVar(&sl.cpuprofileFlagValue, cpuprofileFileFlagName, "", "optional path a cpuprofile file (.prof)")
	flags.StringVar(&sl.memprofileFlagValue, memprofileFileFlagName, "", "optional path a memory profile file (.prof)")
	flags.Var(&sl.symbolProviderNamesFlagValue, scalaSymbolProviderFlagName, "name of a symbol provider implementation to enable")
	flags.Var(&sl.conflictResolverNamesFlagValue, scalaConflictResolverFlagName, "name of a conflict resolver implementation to enable")
	flags.Var(&sl.depsCleanerNamesFlagValue, scalaDepsCleanerFlagName, "name of a deps cleaner implementation to enable")
	flags.Var(&sl.existingScalaBinaryRulesFlagValue, existingScalaBinaryRuleFlagName, "LOAD%NAME mapping for a custom existing scala binary rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scalabinary'")
	flags.Var(&sl.existingScalaLibraryRulesFlagValue, existingScalaLibraryRuleFlagName, "LOAD%NAME mapping for a custom existing scala library rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scala_library'")
	flags.Var(&sl.existingScalaTestRulesFlagValue, existingScalaTestRuleFlagName, "LOAD%NAME mapping for a custom existing scala test rule implementation (e.g. '@io_bazel_rules_scala//scala:scala.bzl%scala_test'")

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
	if sl.debugProcessFlagValue {
		collections.PrintProcessIdForDelveAndWait()
	}

	sl.symbolResolver = newUniverseResolver(sl, sl.globalPackages)

	if err := sl.setupSymbolProviders(flags, c, sl.symbolProviderNamesFlagValue); err != nil {
		return err
	}
	if err := sl.setupConflictResolvers(flags, c, sl.conflictResolverNamesFlagValue); err != nil {
		return err
	}
	if err := sl.setupDepsCleaners(flags, c, sl.depsCleanerNamesFlagValue); err != nil {
		return err
	}
	if err := sl.setupExistingScalaBinaryRules(sl.existingScalaBinaryRulesFlagValue); err != nil {
		return err
	}
	if err := sl.setupExistingScalaLibraryRules(sl.existingScalaLibraryRulesFlagValue); err != nil {
		return err
	}
	if err := sl.setupExistingScalaTestRules(sl.existingScalaTestRulesFlagValue); err != nil {
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

	sl.phaseTransition("generate")

	return nil
}

func (sl *scalaLang) setupSymbolProviders(flags *flag.FlagSet, c *config.Config, names []string) error {
	sl.logger.Debug().Msgf("setting up %d symbol providers", len(names))

	providers, err := resolver.GetNamedSymbolProviders(names)
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
	sl.logger.Debug().Msgf("setting up %d conflict resolvers", len(names))

	for _, name := range names {
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

func (sl *scalaLang) setupDepsCleaners(flags *flag.FlagSet, c *config.Config, names []string) error {
	sl.logger.Debug().Msgf("setting up %d deps cleaners", len(names))

	for _, name := range names {
		cleaner, ok := resolver.GlobalDepsCleanerRegistry().GetDepsCleaner(name)
		if !ok {
			return fmt.Errorf("-%s not found: %q", scalaDepsCleanerFlagName, name)
		}
		if err := cleaner.CheckFlags(flags, c); err != nil {
			return err
		}
		sl.depsCleaners[name] = cleaner
	}
	return nil
}

func (sl *scalaLang) setupExistingScalaBinaryRules(rules []string) error {
	sl.logger.Debug().Msgf("setting up %d existing scala binary rules", len(rules))

	for _, fqn := range rules {
		parts := strings.SplitN(fqn, "%", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid -existing_scala_binary_rule flag value: wanted '%%' separated string, got %q", fqn)
		}
		if err := sl.setupExistingScalaBinaryRule(fqn, parts[0], parts[1]); err != nil {
			return err
		}
	}
	return nil
}

func (sl *scalaLang) setupExistingScalaLibraryRules(rules []string) error {
	sl.logger.Debug().Msgf("setting up %d existing scala library rules", len(rules))

	for _, fqn := range rules {
		parts := strings.SplitN(fqn, "%", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid -existing_scala_library_rule flag value: wanted '%%' separated string, got %q", fqn)
		}
		if err := sl.setupExistingScalaLibraryRule(fqn, parts[0], parts[1]); err != nil {
			return err
		}
	}
	return nil
}

func (sl *scalaLang) setupExistingScalaTestRules(rules []string) error {
	sl.logger.Debug().Msgf("setting up %d existing scala test rules", len(rules))

	for _, fqn := range rules {
		parts := strings.SplitN(fqn, "%", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid -existing_scala_test_rule flag value: wanted '%%' separated string, got %q", fqn)
		}
		if err := sl.setupExistingScalaTestRule(fqn, parts[0], parts[1]); err != nil {
			return err
		}
	}
	return nil
}

func (sl *scalaLang) setupExistingScalaBinaryRule(fqn, load, kind string) error {
	provider := &existingScalaRuleProvider{
		load:      load,
		name:      kind,
		isBinary:  true,
		isLibrary: false,
		isTest:    false,
	}
	return sl.ruleProviderRegistry.RegisterProvider(fqn, provider)
}

func (sl *scalaLang) setupExistingScalaLibraryRule(fqn, load, kind string) error {
	provider := &existingScalaRuleProvider{
		load:      load,
		name:      kind,
		isBinary:  false,
		isLibrary: true,
		isTest:    false,
	}
	return sl.ruleProviderRegistry.RegisterProvider(fqn, provider)
}

func (sl *scalaLang) setupExistingScalaTestRule(fqn, load, kind string) error {
	provider := &existingScalaRuleProvider{
		load:      load,
		name:      kind,
		isBinary:  false,
		isLibrary: false,
		isTest:    true,
	}
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

func (sl *scalaLang) dumpResolvedImportMap() {
	if sl.importsFileFlagValue == "" {
		return
	}
	filename := os.ExpandEnv(sl.importsFileFlagValue)
	if err := sl.writeResolvedImportsMapFile(filename); err != nil {
		log.Fatalf("writing resolved imports: %v", err)
	}
	sl.logger.Debug().Msgf("Wrote resolved import map to: %s", filename)
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
		log.Println("Wrote cpuprofile to", sl.cpuprofileFlagValue)
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
