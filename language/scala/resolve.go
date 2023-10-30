package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Imports implements part of the language.Language interface
func (sl *scalaLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	from := label.Label{Pkg: f.Pkg, Name: r.Name()}

	pkg, ok := sl.packages[from.Pkg]
	if !ok {
		return nil
	}

	provider := pkg.ruleProvider(r)
	if provider == nil {
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
	// gazelle supplies the 'from' label fully-qualified (label.Repo is set to
	// the current workspace name).  However, all the symbols provided are using
	// the default workspace, so normalize it here without the repoName to make
	// matching simpler.
	if from.Repo == c.RepoName {
		from.Repo = ""
	}

	if !sl.isResolvePhase {
		sl.isResolvePhase = true
		sl.onResolve()
	}

	pkg, ok := sl.packages[from.Pkg]
	if !ok {
		return
	}

	if r.Kind() == packageMarkerRuleKind {
		resolvePackageMarkerRule(sl.progress, r, len(sl.packages), sl.wantProgress)
		sl.remainingPackages--
	} else {
		pkg.Resolve(c, ix, rc, r, importsRaw, from)
	}

	if sl.remainingPackages == 0 {
		sl.onEnd()
	}
}
