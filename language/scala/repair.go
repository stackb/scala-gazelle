package scala

import (
	"fmt"
	"log"
	"path"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/stackb/scala-gazelle/pkg/procutil"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
	"github.com/stackb/scala-gazelle/pkg/sweep"
)

type repairMode int

const (
	RepairNone repairMode = iota
	RepairBatch
	RepairWatch
	RepairTransitive
)

// String partially implements the flag.Value interface.
func (i *repairMode) String() string {
	switch *i {
	case RepairNone:
		return "none"
	case RepairBatch:
		return "batch"
	case RepairWatch:
		return "watch"
	case RepairTransitive:
		return "transitive"
	}
	return "unknown"
}

// Set implements the flag.Value interface.
func (i *repairMode) Set(value string) error {
	switch value {
	case "", "none":
		*i = RepairNone
	case "batch":
		*i = RepairBatch
	case "watch":
		*i = RepairWatch
	case "transitive":
		*i = RepairTransitive
	default:
		return fmt.Errorf("unknown repair value: %s", value)
	}
	return nil
}

func (sl *scalaLang) repair() {
	PrintEnv(log.Printf)

	if err := sl.repairDeps(sl.repairMode); err != nil {
		log.Printf("warning: repair failed: %v", err)
	}
}

func (sl *scalaLang) repairDeps(mode repairMode) error {
	switch mode {
	case RepairBatch:
		return sl.repairBatch()
	case RepairWatch:
		return sl.repairWatch()
	case RepairTransitive:
		return sl.repairTransitive()
	default:
		return nil
	}
}

func (sl *scalaLang) repairBatch() error {
	rules := gatherResolvableScalaRuleMap(sl.knownRules)
	imports := makeResolvedImports(sl.globalScope)

	fixer := sweep.NewDepFixer(sl.progress, sl.repoRoot, "", rules, imports.Imports, sl, sl.globalScope)
	return fixer.Batch()
}

func (sl *scalaLang) repairWatch() error {
	dir, ok := procutil.LookupEnv(SCALA_GAZELLE_WATCH_DIR)
	if !ok {
		return fmt.Errorf("error: %v must be set to the directory to watch", SCALA_GAZELLE_WATCH_DIR)
	}
	if !path.IsAbs(dir) {
		dir = path.Join(sl.repoRoot, dir)
	}

	rules := gatherResolvableScalaRuleMap(sl.knownRules)
	imports := makeResolvedImports(sl.globalScope)

	fixer := sweep.NewDepFixer(sl.progress, sl.repoRoot, "", rules, imports.Imports, sl, sl.globalScope)

	return fixer.Watch(dir)
}

func (sl *scalaLang) repairTransitive() error {
	rules := gatherResolvableScalaRuleMap(sl.knownRules)
	imports := makeResolvedImports(sl.globalScope)

	fixer := sweep.NewDepFixer(sl.progress, sl.repoRoot, "", rules, imports.Imports, sl, sl.globalScope)

	return fixer.Transitive()
}

func gatherResolvableScalaRuleMap(knownRules map[label.Label]*rule.Rule) sweep.ResolvableScalaRuleMap {
	scalaRules := make(sweep.ResolvableScalaRuleMap)

	for _, knownRule := range knownRules {
		scalaRule, ok := scalarule.GetRule(knownRule)
		if !ok {
			continue
		}
		resolveFunc := knownRule.PrivateAttr("_scala_resolve_closure").(func())
		scalaRules[scalaRule] = resolveFunc
	}

	return scalaRules
}
