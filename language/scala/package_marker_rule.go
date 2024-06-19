package scala

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/mobyprogress"
)

const packageMarkerRuleKind = "_package_marker"

// generatePackageMarkerRule creates a dummy rule that forces gazelle to run the
// resolve phase at least once per package; used for tracking progress during
// the resolve phase.
func generatePackageMarkerRule(pkgNum int, pkg *scalaPackage) *rule.Rule {
	r := rule.NewRule(packageMarkerRuleKind, packageMarkerRuleKind)
	r.SetPrivateAttr("n", pkgNum)
	r.SetPrivateAttr("pkg", pkg)
	return r
}

// resolvePackageMarkerRule is called when a package marker rule is resolved.
func resolvePackageMarkerRule(output mobyprogress.Output, r *rule.Rule, total int, wantProgress bool) {
	current := r.PrivateAttr("n").(int)
	if wantProgress {
		writeResolveProgress(output, current, total, current == total)
	}

	pkg := r.PrivateAttr("pkg").(*scalaPackage)
	pkg.Finalize()

	r.Delete()
}
