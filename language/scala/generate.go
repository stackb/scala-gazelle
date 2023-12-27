package scala

import (
	"fmt"
	"log"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/emirpasic/gods/maps/linkedhashmap"
)

const debugGenerate = false

// GenerateRules implements part of the language.Language interface
func (sl *scalaLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {

	if args.File == nil {
		return language.GenerateResult{}
	}

	t1 := time.Now()

	if sl.wantProgress && sl.cache.PackageCount > 0 {
		writeGenerateProgress(sl.progress, sl.packages.Size(), int(sl.cache.PackageCount))
	}

	sc := getScalaConfig(args.Config)
	pkg := newScalaPackage(args.Rel, args.File, sc, sl.ruleProviderRegistry, sl.parser, sl)
	sl.packages.Put(args.Rel, pkg)
	sl.remainingPackages++

	rules := pkg.Rules()
	for _, r := range rules {
		from := label.Label{Pkg: args.Rel, Name: r.Name()}
		sl.PutKnownRule(from, r)
	}

	rules = append(rules, generatePackageMarkerRule(sl.packages.Size()))

	if sc.shouldAnnotateGeneration() {
		rules = append(rules, annotateGeneration(args.File, *sl.packages))
	}

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

func annotateGeneration(file *rule.File, packages linkedhashmap.Map) *rule.Rule {
	tags := []string{}
	for i, k := range packages.Keys() {
		tags = append(tags, fmt.Sprintf("%d: %v", i, k))
	}
	r := rule.NewRule("filegroup", "_gazelle_generate")
	r.SetAttr("srcs", []string{"BUILD.bazel"})
	r.SetAttr("tags", tags)
	return r
}
