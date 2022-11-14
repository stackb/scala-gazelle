package scala

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/pcj/moprogress"
)

// GenerateRules implements part of the language.Language interface
func (sl *scalaLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {

	if sl.totalPackageCount > 0 {
		sl.progress.WriteProgress(moprogress.Progress{
			ID:      "walk",
			Action:  "generating rules",
			Total:   int64(sl.totalPackageCount),
			Current: int64(len(sl.packages)),
			Units:   "packages",
		})
	}

	cfg := getOrCreateScalaConfig(args.Config)

	pkg := newScalaPackage(sl.ruleRegistry, sl.scalaFileParser, sl.importRegistry, args.Rel, args.File, cfg)
	// search for child packages, but only assign if a parent has not already
	// been assigned.  Given that gazelle uses a DFS walk, we should assign the
	// child to the nearest parent.
	for rel, child := range sl.packages {
		if child.parent != nil {
			continue
		}
		if !strings.HasPrefix(rel, args.Rel) {
			continue
		}
		child.parent = pkg
	}
	sl.packages[args.Rel] = pkg
	sl.lastPackage = pkg

	rules := pkg.Rules()
	sl.remainingRules += len(rules)
	empty := pkg.Empty()

	imports := make([]interface{}, len(rules))
	for i, r := range rules {
		imports[i] = r.PrivateAttr(config.GazelleImportsKey)
	}

	return language.GenerateResult{
		Gen:     rules,
		Empty:   empty,
		Imports: imports,
	}
}
