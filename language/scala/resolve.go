package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"
	"github.com/stackb/scala-gazelle/pkg/crossresolve"
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
		if r.Kind() == packageMarkerRuleKind {
			resolvePackageMarkerRule(sl.progress, r, len(sl.packages))
		} else {
			pkg.Resolve(c, ix, rc, r, importsRaw, from)
		}

		sl.remainingRules--

		if sl.remainingRules == 0 {
			sl.onEnd()
		}
	} else {
		log.Printf("no known rule package for %v", from.Pkg)
	}

}

// onResolve is called when gazelle transitions from the generate phase to the resolve phase
func (sl *scalaLang) onResolve() {

	for _, r := range sl.resolvers {
		if l, ok := r.(crossresolve.GazellePhaseTransitionListener); ok {
			l.OnResolve()
		}
	}

	sl.scalaCompiler.OnResolve()
}

// onEnd is called when the last rule has been resolved.
func (sl *scalaLang) onEnd() {
	sl.scalaCompiler.stop()
	// sl.recordDeps()
	if len(sl.packages) != sl.totalPackageCount {
		mobyprogress.Messagef(
			sl.progress,
			"generate", "expected %d packages, visited %d (update -total_package_count=%d to suppress this message)",
			sl.totalPackageCount,
			len(sl.packages),
			len(sl.packages))
	}
}
