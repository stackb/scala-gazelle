package scala

import (
	"os"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"
	"github.com/stackb/rules_proto/pkg/protoc"

	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalacompile"
	"github.com/stackb/scala-gazelle/pkg/scalaparse"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

const scalaLangName = "scala"

// scalaLang implements language.Language.
type scalaLang struct {
	// cacheFileFlagValue is the main cache file, if enabled
	cacheFileFlagValue string
	// symbolProviderNamesFlagValue is a repeatable list of resolver to enable
	symbolProviderNamesFlagValue collections.StringSlice
	// conflictResolverNamesFlagValue is a repeatable list of conflict resolver
	// to enable
	conflictResolverNamesFlagValue collections.StringSlice
	// existingScalaRulesFlagValue is the value of the existing_scala_rule
	// repeatable flag
	existingScalaRulesFlagValue collections.StringSlice
	cpuprofileFlagValue         string
	memprofileFlagValue         string
	// cache is the loaded cache, if configured
	cache *scpb.Cache
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
	// knownScalaRules is a map of all scala rules protos
	knownScalaRules map[label.Label]*sppb.Rule
	// conflictResolvers is a map of all known generated rules
	conflictResolvers map[string]resolver.ConflictResolver
	// globalScope includes all known symbols in the universe
	globalScope resolver.Scope
	// symbolProviders is a list of providers
	symbolProviders []resolver.SymbolProvider
	// symbolResolver is our top-level known import resolver implementation
	symbolResolver resolver.SymbolResolver
	// compiler is the compiler tool
	compiler scalacompile.Compiler
	// sourceProvider is the sourceProvider implementation.
	sourceProvider *provider.SourceProvider
	// parser is the parser instance
	parser scalaparse.Parser
}

// Name implements part of the language.Language interface
func (sl *scalaLang) Name() string { return scalaLangName }

// KnownDirectives implements part of the language.Language interface
func (*scalaLang) KnownDirectives() []string {
	return []string{
		scalaRuleDirective,
		resolveGlobDirective,
		resolveWithDirective,
		scalaAnnotateDirective,
		resolveKindRewriteNameDirective,
	}
}

// NewLanguage is called by Gazelle to install this language extension in a
// binary.
func NewLanguage() language.Language {
	packages := make(map[string]*scalaPackage)

	lang := &scalaLang{
		cache:                &scpb.Cache{},
		globalScope:          resolver.NewTrieScope(),
		knownRules:           make(map[label.Label]*rule.Rule),
		knownScalaRules:      make(map[label.Label]*sppb.Rule),
		conflictResolvers:    make(map[string]resolver.ConflictResolver),
		packages:             packages,
		progress:             mobyprogress.NewProgressOutput(mobyprogress.NewOut(os.Stderr)),
		ruleProviderRegistry: scalarule.GlobalProviderRegistry(),
	}

	lang.sourceProvider = provider.NewSourceProvider(func(msg string) {
		writeParseProgress(lang.progress, msg)
	})
	lang.parser = scalaparse.NewMemoParser(lang.GetKnownScalaRule, lang.sourceProvider)

	scalac := scalacompile.NewScalacCompiler()
	lang.compiler = scalacompile.NewMemoCompiler(scalac)

	lang.AddSymbolProvider(lang.sourceProvider)
	lang.AddSymbolProvider(scalac)
	lang.AddSymbolProvider(provider.NewJavaProvider())
	lang.AddSymbolProvider(provider.NewMavenProvider(scalaLangName))
	lang.AddSymbolProvider(provider.NewProtobufProvider(scalaLangName, scalaLangName, protoc.GlobalResolver().Provided))

	return lang
}
