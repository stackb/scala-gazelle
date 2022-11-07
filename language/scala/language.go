package scala

import (
	"os"

	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/stackb/scala-gazelle/pkg/progress"
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
	sourceResolver := newScalaSourceIndexResolver(depends)
	classResolver := newScalaClassIndexResolver(depends)
	mavenResolver := newMavenResolver()
	scalaCompiler := newScalaCompiler()
	// var scalaCompiler *scalaCompiler
	importRegistry = newImportRegistry(sourceResolver, classResolver, scalaCompiler)
	vizServer := newGraphvizServer(packages, importRegistry)

	out := progress.NewOut(os.Stderr)

	return &scalaLang{
		ruleRegistry:    globalRuleRegistry,
		scalaFileParser: sourceResolver,
		scalaCompiler:   scalaCompiler,
		packages:        packages,
		importRegistry:  importRegistry,
		resolvers: []ConfigurableCrossResolver{
			sourceResolver,
			classResolver,
			mavenResolver,
		},
		progress: progress.NewProgressOutput(out),
		viz:      vizServer,
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
	// resolvers is a list of cross resolver implementations.  Typically there
	// are two: one to help with third-party code, one to help with first-party
	// code.
	resolvers []ConfigurableCrossResolver
	// viz is the dependency vizualization engine
	viz *graphvizServer
	// lastPackage tracks if this is the last generated package
	lastPackage *scalaPackage
	// totalPackageCount is used for progress
	totalPackageCount int
	// remainingRules is a counter that tracks when all rules have been resolved.
	remainingRules int
	// totalRules is used for progress
	totalRules int
	// progress is the progress interface
	progress progress.Output
}

// Name implements part of the language.Language interface
func (sl *scalaLang) Name() string { return ScalaLangName }
