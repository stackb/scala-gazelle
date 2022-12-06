package scala

import (
	"os"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"

	"github.com/stackb/rules_proto/pkg/protoc"
	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const scalaLangName = "scala"

// scalaLang implements language.Language.
type scalaLang struct {
	// cacheFileFlagValue is the main cache file, if enabled
	cacheFileFlagValue string
	// importProviderNamesFlagValue is a repeatable list of resolver to enable
	importProviderNamesFlagValue collections.StringSlice
	// scalaExistingRulesFlagValue is the value of the scala_existing_rule repeatable flag
	scalaExistingRulesFlagValue collections.StringSlice
	// cache is the loaded cache, if configured
	cache *scpb.Cache
	// ruleRegistry is the rule registry implementation.  This holds the rules
	// configured via gazelle directives by the user.
	ruleRegistry RuleRegistry
	// sourceProvider is the source resolver implementation.
	sourceProvider *provider.ScalaparseProvider
	// packages is map from the config.Rel to *scalaPackage for the
	// workspace-relative package name.
	packages map[string]*scalaPackage
	// isResolvePhase is a flag that is tracks if at least one Resolve() call
	// has occurred.  It can be used to determine when the rule indexing phase
	// has completed and deps resolution phase has started (it calls
	// onResolvePhase).
	isResolvePhase bool
	// lastPackage tracks if this is the last generated package
	lastPackage *scalaPackage
	// remainingRules is a counter that tracks when all rules have been resolved.
	remainingRules int
	// totalRules is used for progress
	totalRules int
	// progress is the progress interface
	progress mobyprogress.Output
	// knownRules is a map of all known generated rules
	knownRules map[label.Label]*rule.Rule
	// knownImports is a map of all known generated import providers
	knownImports resolver.KnownImportRegistry
	// knownImportProviders is a list of providers
	knownImportProviders []resolver.KnownImportProvider
	// knownImportResolver is our top-level known import resolver implementation
	knownImportResolver resolver.KnownImportResolver
}

// Name implements part of the language.Language interface
func (sl *scalaLang) Name() string { return scalaLangName }

// KnownDirectives implements part of the language.Language interface
func (*scalaLang) KnownDirectives() []string {
	return []string{
		ruleDirective,
		resolveGlobDirective,
		resolveWithDirective,
		scalaAnnotateDirective,
		resolveKindRewriteName,
	}
}

// NewLanguage is called by Gazelle to install this language extension in a
// binary.
func NewLanguage() language.Language {
	packages := make(map[string]*scalaPackage)

	lang := &scalaLang{
		cache:        &scpb.Cache{},
		knownImports: resolver.NewKnownImportRegistryTrie(),
		knownRules:   make(map[label.Label]*rule.Rule),
		packages:     packages,
		progress:     mobyprogress.NewProgressOutput(mobyprogress.NewOut(os.Stderr)),
		ruleRegistry: globalRuleRegistry,
	}

	lang.sourceProvider = provider.NewScalaparseProvider(func(msg string) {
		writeParseProgress(lang.progress, msg)
	})

	lang.AddKnownImportProvider(lang.sourceProvider)
	lang.AddKnownImportProvider(provider.NewJarIndexProvider())
	lang.AddKnownImportProvider(provider.NewRulesJvmExternalProvider(scalaLangName))
	lang.AddKnownImportProvider(provider.NewStackbRulesProtoProvider(scalaLangName, scalaLangName, protoc.GlobalResolver()))

	return lang
}
