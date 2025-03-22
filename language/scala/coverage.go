package scala

import "os"

type packageRuleCoverage struct {
	// managed represents the total number of rules that are managed by
	// scala-gazelle (actual number of rules that we provided deps for)
	managed int
	// total represents the total number of rules in a package that we have a
	// RuleProvider for.
	total int
}

func (sl *scalaLang) reportCoverage(printf func(format string, v ...any)) {
	if !sl.existingScalaRuleCoverageFlagValue {
		return
	}

	var managed int
	var total int
	for _, pkg := range sl.packages {
		managed += pkg.ruleCoverage.managed
		total += pkg.ruleCoverage.total
	}

	percent := float32(managed) / float32(total) * 100

	printCoverage := true
	if val, ok := os.LookupEnv("SCALA_GAZELLE_SHOW_COVERAGE"); ok {
		if val == "false" || val == "0" {
			printCoverage = false
		}
	}
	if printCoverage {
		printf("scala-gazelle coverage is %0.1f%% (%d/%d)", percent, managed, total)
	}
}
