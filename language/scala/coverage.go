package scala

type packageRuleCoverage struct {
	// managed represents the total number of rules that are managed by
	// scala-gazelle (actual number of rules that we provided deps for)
	managed int
	// total represents the total number of rules in a package that we have a
	// RuleProvider for.
	total int
}

func (sl *scalaLang) reportCoverage(printf func(format string, v ...any)) {
	if !sl.reportCoverageFlagValue {
		return
	}

	var managed int
	var total int

	for _, pkg := range sl.packages {
		coverage := pkg.ruleCoverage()
		managed += coverage.managed
		total += coverage.total
	}

	percent := float32(managed) / float32(total) * 100

	printf("scala-gazelle coverage is %0.1f%% (%d/%d)", percent, managed, total)
}
