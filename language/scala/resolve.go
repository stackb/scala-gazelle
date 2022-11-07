package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/rules_proto/pkg/protoc"
	"github.com/stackb/scala-gazelle/pkg/progress"
)

// Imports implements part of the language.Language interface
func (sl *scalaLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	from := label.New("", f.Pkg, r.Name())

	pkg, ok := sl.packages[from.Pkg]
	if !ok {
		// log.Println("scala.Imports(): Unknown package", from.Pkg)
		return nil
	}

	provider := pkg.ruleProvider(r)
	// NOTE: gazelle attempts to index rules found in the build file regardless
	// of whether we returned the rule from GenerateRules or not, so this will
	// be nil in that case.
	if provider == nil {
		// log.Println("scala.Imports(): Unknown provider", from)
		return nil
	}

	return provider.Imports(c, r, f)
}

// Embeds implements part of the language.Language interface
func (*scalaLang) Embeds(r *rule.Rule, from label.Label) []label.Label { return nil }

// Resolve implements part of the language.Language interface
func (sl *scalaLang) Resolve(
	c *config.Config,
	ix *resolve.RuleIndex,
	rc *repo.RemoteCache,
	r *rule.Rule,
	importsRaw interface{},
	from label.Label,
) {
	if !sl.isResolvePhase {
		sl.isResolvePhase = true
		sl.onResolve()
		sl.totalRules = sl.remainingRules
	}

	if pkg, ok := sl.packages[from.Pkg]; ok {
		pkg.Resolve(c, ix, rc, r, importsRaw, from)

		sl.remainingRules--

		if sl.remainingRules == 0 {
			sl.onEnd()
		}
	} else {
		log.Printf("no known rule package for %v", from.Pkg)
	}

	if sl.totalRules > 0 {
		sl.progress.WriteProgress(progress.Progress{
			ID:      "resolve",
			Action:  "resolving dependencies",
			Total:   int64(sl.totalRules),
			Current: int64(sl.totalRules - sl.remainingRules),
			Units:   "rules",
		})
	}
}

// onResolve is called when gazelle transitions from the generate phase to the resolve phase
func (sl *scalaLang) onResolve() {

	for _, r := range sl.resolvers {
		if l, ok := r.(GazellePhaseTransitionListener); ok {
			l.OnResolve()
		}
	}

	sl.scalaCompiler.OnResolve()
	sl.viz.OnResolve()

	// gather proto imports
	for from, imports := range protoc.GlobalResolver().Provided(ScalaLangName, ScalaLangName) {
		sl.importRegistry.Provides(from, imports)
	}

	// gather 1p/3p imports
	for _, rslv := range sl.resolvers {
		if ip, ok := rslv.(protoc.ImportProvider); ok {
			for from, imports := range ip.Provided(ScalaLangName, ScalaLangName) {
				sl.importRegistry.Provides(from, imports)
			}
		}
	}

	sl.importRegistry.OnResolve()
}

// onEnd is called when the last rule has been resolved.
func (sl *scalaLang) onEnd() {
	sl.scalaCompiler.stop()
	// sl.recordDeps()
	sl.viz.OnEnd()
}

// recordDeps writes deps info to the graph once all rules resolved.
func (sl *scalaLang) recordDeps() {
	for _, pkg := range sl.packages {
		for _, r := range pkg.rules {
			from := label.New("", pkg.rel, r.Name())
			for _, dep := range r.AttrStrings("deps") {
				to, err := label.Parse(dep)
				if err != nil {
					continue
				}
				sl.importRegistry.AddDependency("rule/"+from.String(), "rule/"+to.String(), "depends")
			}
		}
	}
}