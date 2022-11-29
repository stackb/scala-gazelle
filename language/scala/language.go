package scala

import (
	"os"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"
	"github.com/stackb/rules_proto/pkg/protoc"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

const ScalaLangName = "scala"

// NewLanguage is called by Gazelle to install this language extension in a
// binary.
func NewLanguage() language.Language {
	var importRegistry *importRegistry
	depends := func(src, dst, kind string) {
		importRegistry.AddDependency(src, dst, kind)
	}
	packages := make(map[string]*scalaPackage)

	scalaCompiler := newScalaCompiler()
	// var scalaCompiler *scalaCompiler

	classResolver := newScalaClassIndexResolver(depends)
	sourceResolver := crossresolve.NewScalaSourceCrossResolver(ScalaLangName, depends)
	protoResolver := crossresolve.NewProtoResolver(ScalaLangName, protoc.GlobalResolver().Provided)
	mavenResolver := crossresolve.NewMavenResolver("java")
	jarResolver := crossresolve.NewJarIndexCrossResolver(ScalaLangName, depends)

	importRegistry = newImportRegistry(sourceResolver, classResolver, scalaCompiler)

	crossresolve.Resolvers().MustRegisterResolver("source", sourceResolver)
	crossresolve.Resolvers().MustRegisterResolver("maven", mavenResolver)
	crossresolve.Resolvers().MustRegisterResolver("proto", protoResolver)
	crossresolve.Resolvers().MustRegisterResolver("jar", jarResolver)

	return &scalaLang{
		ruleRegistry:    globalRuleRegistry,
		scalaFileParser: sourceResolver,
		scalaCompiler:   scalaCompiler,
		packages:        packages,
		importRegistry:  importRegistry,
		resolvers:       make(map[string]crossresolve.ConfigurableCrossResolver),
		progress:        mobyprogress.NewProgressOutput(mobyprogress.NewOut(os.Stderr)),
		allRules:        make(map[label.Label]*rule.Rule),
	}
}

// scalaLang implements language.Language.
type scalaLang struct {
	// ruleRegistry is the rule registry implementation.  This holds the rules
	// configured via gazelle directives by the user.
	ruleRegistry RuleRegistry
	// importRegistry instance tracks all known info about imports and rules and
	// is used during import disambiguation.
	importRegistry *importRegistry
	// scalaFileParser is the parser implementation.  This is given to each
	// ScalaPackage during GenerateRules such that rule implementations can use
	// it.
	scalaFileParser ScalaFileParser
	// scalaCompiler is the compiler implementation.  This is passed to the
	// importRegistry for use during import disambiguation.
	scalaCompiler *scalaCompiler
	// packages is map from the config.Rel to *scalaPackage for the
	// workspace-relative packate name.
	packages map[string]*scalaPackage
	// isResolvePhase is a flag that is tracks if at least one Resolve() call
	// has occurred.  It can be used to determine when the rule indexing phase
	// has completed and deps resolution phase has started (it calls
	// onResolvePhase).
	isResolvePhase bool
	// resolvers is a list of cross resolver implementations named by the -scala_resolvers flag
	resolvers map[string]crossresolve.ConfigurableCrossResolver
	// lastPackage tracks if this is the last generated package
	lastPackage *scalaPackage
	// resolverNames is a comma-separated list of resolver to enable
	resolverNames string
	// totalPackageCount is used for progress
	totalPackageCount int
	// remainingRules is a counter that tracks when all rules have been resolved.
	remainingRules int
	// totalRules is used for progress
	totalRules int
	// progress is the progress interface
	progress mobyprogress.Output
	// scalaExistingRules is the value of the scala_existing_rule repeatable flag
	scalaExistingRules stringSliceFlags
	// ruleIndex is a map of all known generated rules
	allRules map[label.Label]*rule.Rule
}

// Name implements part of the language.Language interface
func (sl *scalaLang) Name() string { return ScalaLangName }

// KnownDirectives implements part of the language.Language interface
func (*scalaLang) KnownDirectives() []string {
	return []string{
		ruleDirective,
		overrideDirective,
		indirectDependencyDirective,
		implicitImportDirective,
		scalaExplainDependencies,
		mapKindImportNameDirective,
	}
}
