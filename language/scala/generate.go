package scala

import (
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
)

// GenerateRules implements part of the language.Language interface
func (sl *scalaLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	if args.File == nil {
		return language.GenerateResult{}
	}

	if sl.totalPackageCount > 0 {
		writeGenerateProgress(sl.progress, len(sl.packages), sl.totalPackageCount)
	}

	cfg := getOrCreateScalaConfig(sl, args.Config, args.Rel)

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
		sl.importRegistry.AddDependency("pkg/"+args.Rel, "pkg/"+rel, "pkg")
	}
	sl.packages[args.Rel] = pkg
	sl.importRegistry.AddDependency("ws/default", "pkg/"+args.Rel, "ws")
	sl.lastPackage = pkg

	rules := pkg.Rules()

	rules = append(rules, generatePackageMarkerRule(len(sl.packages)))

	sl.remainingRules += len(rules)

	imports := make([]interface{}, len(rules))
	for i, r := range rules {
		imports[i] = r.PrivateAttr(config.GazelleImportsKey)
		sl.importRegistry.AddDependency("pkg/"+args.Rel, "rule/"+label.New("", args.Rel, r.Name()).String(), "rule")
		if r.Kind() != packageMarkerRuleKind {
			from := label.New(args.Config.RepoName, args.Rel, r.Name())
			sl.setGlobalRule(from, r)
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}
