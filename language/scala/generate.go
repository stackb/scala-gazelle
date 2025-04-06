package scala

import (
	"log"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
)

const debugGenerate = false

// GenerateRules implements part of the language.Language interface
func (sl *scalaLang) GenerateRules(args language.GenerateArgs) (result language.GenerateResult) {
	now := time.Now()

	sl.logger.Debug().Msgf("visiting directory %s", args.Rel)

	sc := scalaconfig.Get(args.Config)
	if args.File == nil && !sc.GenerateBuildFiles() {
		return
	}

	if sl.wantProgress && sl.cache.PackageCount > 0 {
		writeGenerateProgress(sl.progress, len(sl.packages), int(sl.cache.PackageCount))
	}

	logger := sl.logger.With().Str("rel", args.Rel).Logger()
	pkg := newScalaPackage(logger, args, sc, sl.ruleProviderRegistry, sl.parser, sl)
	sl.packages[args.Rel] = pkg
	sl.remainingPackages++

	rules := pkg.Rules()
	for _, r := range rules {
		from := label.Label{Pkg: args.Rel, Name: r.Name()}
		sl.PutKnownRule(from, r)
	}

	rules = append(rules, generatePackageMarkerRule(len(sl.packages), pkg))

	imports := make([]interface{}, len(rules))
	for i, r := range rules {
		imports[i] = r.PrivateAttr(config.GazelleImportsKey)
	}

	if debugGenerate {
		t2 := time.Since(now).Round(1 * time.Millisecond)
		if len(rules) > 1 {
			log.Printf("Visited %q (%d rules, %v)", args.Rel, len(rules)-1, t2)
		}
	}

	logger.Debug().Msgf("generated %d rules in %v", len(rules), time.Since(now))

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
		Empty:   pkg.Empty(),
	}
}
