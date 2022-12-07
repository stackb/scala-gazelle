package scala

import (
	"log"
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

	if len(sl.packages) == 0 {
		if err := sl.onGenerate(); err != nil {
			log.Fatal(err)
		}
	}

	if sl.cache.PackageCount > 0 {
		writeGenerateProgress(sl.progress, len(sl.packages), int(sl.cache.PackageCount))
	}

	sc := getOrCreateScalaConfig(args.Config, args.Rel, sl)
	pkg := newScalaPackage(args.Rel, args.File, sc, sl.ruleRegistry, sl.sourceProvider, sl)

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
	for _, r := range rules {
		from := label.New(args.Config.RepoName, args.Rel, r.Name())
		sl.PutKnownRule(from, r)
	}

	rules = append(rules, generatePackageMarkerRule(len(sl.packages)))
	sl.remainingRules += len(rules)

	imports := make([]interface{}, len(rules))
	for i, r := range rules {
		imports[i] = r.PrivateAttr(config.GazelleImportsKey)
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}
