package scala

import (
	"log"

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

	sc := getScalaConfig(args.Config)
	pkg := newScalaPackage(args.Rel, args.File, sc, sl.ruleRegistry, sl.sourceProvider, sl)
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

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}
