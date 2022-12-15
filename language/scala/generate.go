package scala

import (
	"log"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
)

const debugGenerate = false

// GenerateRules implements part of the language.Language interface
func (sl *scalaLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {

	if args.File == nil {
		return language.GenerateResult{}
	}

	t1 := time.Now()

	if sl.cache.PackageCount > 0 {
		writeGenerateProgress(sl.progress, len(sl.packages), int(sl.cache.PackageCount))
	}

	sc := getScalaConfig(args.Config)
	pkg := newScalaPackage(args.Rel, args.File, sc, sl.ruleProviderRegistry, sl.sourceProvider, sl)
	sl.packages[args.Rel] = pkg
	sl.remainingPackages++

	rules := pkg.Rules()
	for _, r := range rules {
		from := label.New(args.Config.RepoName, args.Rel, r.Name())
		sl.PutKnownRule(from, r)
	}

	rules = append(rules, generatePackageMarkerRule(len(sl.packages)))

	imports := make([]interface{}, len(rules))
	for i, r := range rules {
		imports[i] = r.PrivateAttr(config.GazelleImportsKey)
	}

	if debugGenerate {
		t2 := time.Since(t1).Round(1 * time.Millisecond)
		if len(rules) > 1 {
			log.Printf("Visited %q (%d rules, %v)", args.Rel, len(rules)-1, t2)
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}
