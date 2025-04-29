package scala

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/procutil"
)

type packageRuleCoverage struct {
	// managed represents the total number of rules that are managed by
	// scala-gazelle (actual number of rules that we provided deps for)
	managed int
	// total represents the total number of rules in a package that we have a
	// RuleProvider for.
	total int
	// kinds represents the kinds of rules covered, and how many each has
	kinds map[string]int
}

func (sl *scalaLang) reportCoverage(printf func(format string, v ...any)) {
	if !sl.existingScalaRuleCoverageFlagValue {
		return
	}

	kindTotals := make(map[string]int)

	var managed int
	var total int
	for _, pkg := range sl.packages {
		managed += pkg.ruleCoverage.managed
		total += pkg.ruleCoverage.total
		for k, v := range pkg.ruleCoverage.kinds {
			kindTotals[k] += v
		}
	}

	kinds := make([]string, 0, len(kindTotals))
	for kind := range kindTotals {
		kinds = append(kinds, fmt.Sprintf("%s: %d", kind, kindTotals[kind]))
	}
	sort.Strings(kinds)
	totals := strings.Join(kinds, ", ")

	var percent float32
	if total > 0 {
		percent = float32(managed) / float32(total) * 100
	}

	if procutil.LookupBoolEnv(SCALA_GAZELLE_SHOW_COVERAGE, true) {
		printf("scala-gazelle coverage is %0.1f%% (%d/%d) %s", percent, managed, total, totals)
	}
}
