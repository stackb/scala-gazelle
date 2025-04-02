package scala

import (
	"os"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"
	"github.com/rs/zerolog"
	"github.com/stackb/rules_proto/pkg/protoc"

	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/procutil"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	_ "github.com/stackb/scala-gazelle/pkg/scalafiles"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const scalaLangName = "scala"

// scalaLang implements language.Language.
type scalaLang struct {
	// debugProcessFlagValue halts processing and prints the PID for attaching a
	// delve debugger.
	debugProcessFlagValue bool
	// optional flag to print the cache key version
	printCacheKey bool
	// wantProgress is a flag that prints docker style progress messages if enabled
	wantProgress bool
	// cacheFileFlagValue is the main cache file, if enabled
	cacheFileFlagValue string
	// cacheKeyFlagValue is the main cache key, if enabled
	cacheKeyFlagValue string
	// importsFileFlagValue is the name of a file to dump resolved import map to, if enabled
	importsFileFlagValue string
	// symbolProviderNamesFlagValue is a repeatable list of resolver to enable
	symbolProviderNamesFlagValue collections.StringSlice
	// conflictResolverNamesFlagValue is a repeatable list of conflict resolver
	// to enable
	conflictResolverNamesFlagValue collections.StringSlice
	// depsCleanerNamesFlagValue is a repeatable list of deps cleaners
	// to enable
	depsCleanerNamesFlagValue collections.StringSlice
	// existingScalaLibraryRulesFlagValue is the value of the
	// existing_scala_binary_rule repeatable flag
	existingScalaBinaryRulesFlagValue collections.StringSlice
	// existingScalaLibraryRulesFlagValue is the value of the
	// existing_scala_library_rule repeatable flag
	existingScalaLibraryRulesFlagValue collections.StringSlice
	// existingScalaLibraryRulesFlagValue is the value of the
	// existing_scala_test_rule repeatable flag
	existingScalaTestRulesFlagValue    collections.StringSlice
	cpuprofileFlagValue                string
	existingScalaRuleCoverageFlagValue bool
	memprofileFlagValue                string
	// cache is the loaded cache, if configured
	cache scpb.Cache
	// ruleProviderRegistry is the rule registry implementation.  This holds the
	// rules configured via gazelle directives by the user.
	ruleProviderRegistry scalarule.ProviderRegistry
	// packages is map from the config.Rel to *scalaPackage for the
	// workspace-relative package name.
	packages map[string]*scalaPackage
	// isResolvePhase is a flag that is tracks if at least one Resolve() call
	// has occurred.  It can be used to determine when the rule indexing phase
	// has completed and deps resolution phase has started (it calls
	// onResolvePhase).
	isResolvePhase bool
	// remainingPackages is a counter that tracks when all packages have been
	// resolved.
	remainingPackages int
	// progress is the progress interface
	progress mobyprogress.Output
	// knownRules is a map of all known generated rules
	knownRules map[label.Label]*rule.Rule
	// conflictResolvers is a map of all known conflict resolver implementations
	conflictResolvers map[string]resolver.ConflictResolver
	// depsCleaners is a map of all known deps cleaner implementations
	depsCleaners map[string]resolver.DepsCleaner
	// globalScope includes all known symbols in the universe (minus package
	// symbols)
	globalScope resolver.Scope
	// globalPackages is the subset of "globalScope" that are package symbols.
	// We resolve these only after everything else has failed.
	globalPackages resolver.Scope
	// symbolProviders is a list of providers
	symbolProviders []resolver.SymbolProvider
	// symbolResolver is our top-level known import resolver implementation
	symbolResolver resolver.SymbolResolver
	// sourceProvider is the sourceProvider implementation.
	sourceProvider *provider.SourceProvider
	// parser is the parser instance
	parser *parser.MemoParser
	// logFileName is the name of the log file
	// logFile is the open log
	logFile *os.File
	// logger instance
	logger zerolog.Logger
}

// NewLanguage is called by Gazelle to install this language extension in a
// binary.
func NewLanguage() language.Language {
	logFile, logger := setupLogger()

	lang := &scalaLang{
		wantProgress:         wantProgress(),
		cache:                scpb.Cache{},
		globalScope:          resolver.NewTrieScope(),
		globalPackages:       resolver.NewTrieScope(),
		knownRules:           make(map[label.Label]*rule.Rule),
		conflictResolvers:    make(map[string]resolver.ConflictResolver),
		depsCleaners:         make(map[string]resolver.DepsCleaner),
		packages:             make(map[string]*scalaPackage),
		progress:             mobyprogress.NewProgressOutput(mobyprogress.NewOut(os.Stderr)),
		ruleProviderRegistry: scalarule.GlobalProviderRegistry(),
		logFile:              logFile,
		logger:               logger,
	}

	lang.phaseTransition("initialize")

	progress := func(msg string) {
		if lang.wantProgress {
			writeParseProgress(lang.progress, msg)
		}
	}

	lang.sourceProvider = provider.NewSourceProvider(logger.With().Str("provider", "source").Logger(), progress)
	semanticProvider := provider.NewSemanticdbProvider(lang.sourceProvider)
	lang.parser = parser.NewMemoParser(semanticProvider)
	javaProvider := provider.NewJavaProvider()

	lang.AddSymbolProvider(lang.sourceProvider)
	lang.AddSymbolProvider(semanticProvider)
	lang.AddSymbolProvider(javaProvider)
	lang.AddSymbolProvider(provider.NewMavenProvider(scalaLangName))
	lang.AddSymbolProvider(provider.NewProtobufProvider(scalaLangName, scalaLangName, protoc.GlobalResolver().Provided))

	pdcr := resolver.NewPreferredDepsConflictResolver("preferred_deps", javaProvider.GetPreferredDeps())
	resolver.GlobalConflictResolverRegistry().PutConflictResolver(pdcr.Name(), pdcr)

	return lang
}

// Name implements part of the language.Language interface
func (sl *scalaLang) Name() string { return scalaLangName }

// KnownDirectives implements part of the language.Language interface
func (*scalaLang) KnownDirectives() []string {
	return scalaconfig.DirectiveNames()
}

func wantProgress() bool {
	return procutil.LookupBoolEnv("SCALA_GAZELLE_SHOW_PROGRESS", false)
}

func getLoggerFilename() string {
	if name, ok := procutil.LookupEnv(SCALA_GAZELLE_LOG_FILE); ok && len(name) > 0 {
		return name
	}
	if tmpdir, ok := procutil.LookupEnv(TEST_TMPDIR); ok && procutil.DirExists(tmpdir) {
		return filepath.Join(tmpdir, "scala-gazelle.log")
	}

	return ""
}

func setupLogger() (*os.File, zerolog.Logger) {
	filename := getLoggerFilename()

	if filename == "" {
		return nil, zerolog.Nop()
	}

	file, err := os.Create(filename)
	if err != nil {
		panic("cannot open log file: " + err.Error())
	}

	logger := zerolog.New(file).With().Caller().Logger()

	return file, logger
}
