package scala

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pcj/moprogress"
)

const packageMarkerRuleKind = "_package_marker"

func writeGenerateProgress(output moprogress.Output, current, total int) {
	output.WriteProgress(moprogress.Progress{
		ID:      "walk",
		Action:  "generating rules",
		Current: int64(current),
		Total:   int64(total),
		Units:   "packages",
	})
}

func writeResolveProgress(output moprogress.Output, current, total int, lastUpdate bool) {
	output.WriteProgress(moprogress.Progress{
		ID:         "resolve",
		Action:     "resolving dependencies",
		Current:    int64(current),
		Total:      int64(total),
		Units:      "packages",
		LastUpdate: lastUpdate,
	})
}

// generatePackageMarkerRule creates a dummy rule that forces gazelle to run the
// resolve phase at least once per package; used for tracking progress during
// the resolve phase.
func generatePackageMarkerRule(pkgNum int) *rule.Rule {
	r := rule.NewRule(packageMarkerRuleKind, packageMarkerRuleKind)
	r.SetPrivateAttr("n", pkgNum)
	return r
}

// resolvePackageMarkerRule is called when a package marker rule is resolved.
func resolvePackageMarkerRule(output moprogress.Output, r *rule.Rule, total int) {
	current := r.PrivateAttr("n").(int)
	writeResolveProgress(output, current, total, current == total)
	r.Delete()
}
