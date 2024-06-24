package scala

import (
	"os"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"
	"github.com/stackb/rules_proto/pkg/protoc"

	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/parser"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
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
	// symbolProviderNamesFlagValue is a repeatable list of resolver to enable
	symbolProviderNamesFlagValue collections.StringSlice
	// conflictResolverNamesFlagValue is a repeatable list of conflict resolver
	// to enable
	conflictResolverNamesFlagValue collections.StringSlice
	// existingScalaLibraryRulesFlagValue is the value of the
	// existing_scala_binary_rule repeatable flag
	existingScalaBinaryRulesFlagValue collections.StringSlice
	// existingScalaLibraryRulesFlagValue is the value of the
	// existing_scala_library_rule repeatable flag
	existingScalaLibraryRulesFlagValue collections.StringSlice
	// existingScalaLibraryRulesFlagValue is the value of the
	// existing_scala_test_rule repeatable flag
	existingScalaTestRulesFlagValue collections.StringSlice
	cpuprofileFlagValue             string
	memprofileFlagValue             string
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
	// knownFiles is map of the parent File for knownRules
	knownFiles map[string]*rule.File
	// conflictResolvers is a map of all known generated rules
	conflictResolvers map[string]resolver.ConflictResolver
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
}

// Name implements part of the language.Language interface
func (sl *scalaLang) Name() string { return scalaLangName }

// KnownDirectives implements part of the language.Language interface
func (*scalaLang) KnownDirectives() []string {
	return scalaconfig.DirectiveNames()
}

// NewLanguage is called by Gazelle to install this language extension in a
// binary.
func NewLanguage() language.Language {
	packages := make(map[string]*scalaPackage)

	lang := &scalaLang{
		wantProgress:         wantProgress(),
		cache:                scpb.Cache{},
		globalScope:          resolver.NewTrieScope(),
		globalPackages:       resolver.NewTrieScope(),
		knownRules:           make(map[label.Label]*rule.Rule),
		knownFiles:           make(map[string]*rule.File),
		conflictResolvers:    make(map[string]resolver.ConflictResolver),
		packages:             packages,
		progress:             mobyprogress.NewProgressOutput(mobyprogress.NewOut(os.Stderr)),
		ruleProviderRegistry: scalarule.GlobalProviderRegistry(),
	}

	lang.sourceProvider = provider.NewSourceProvider(func(msg string) {
		if lang.wantProgress {
			writeParseProgress(lang.progress, msg)
		}
	})
	lang.parser = parser.NewMemoParser(parser.NewWildcardExpandingParser(lang.sourceProvider))

	lang.AddSymbolProvider(lang.sourceProvider)
	lang.AddSymbolProvider(provider.NewJavaProvider())
	lang.AddSymbolProvider(provider.NewMavenProvider(scalaLangName))
	lang.AddSymbolProvider(provider.NewProtobufProvider(scalaLangName, scalaLangName, protoc.GlobalResolver().Provided))

	return lang
}

func wantProgress() bool {
	if val, ok := os.LookupEnv("SCALA_GAZELLE_SHOW_PROGRESS"); ok {
		switch strings.ToLower(val) {
		case "true", "1":
			return true
		case "false", "0":
			return false
		}
	}
	// default to true
	return true
}
