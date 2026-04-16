package maven

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/stackb/scala-gazelle/pkg/bazel"
	"github.com/stackb/scala-gazelle/pkg/resolver"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

type Resolver interface {
	Name() string
	Resolve(pkg string) (label.Label, error)
}

// mavenResolver finds Maven provided packages by reading the maven_install.json
// file from rules_jvm_external.
type mavenResolver struct {
	lang      string
	name      string
	warn      warnFunc
	data      *StringMultiSet
	artifacts map[string]label.Label
}

type warnFunc func(format string, args ...interface{})
type putSymbolFunc func(*resolver.Symbol) error

func NewResolver(installFile, mavenWorkspaceName, lang string, warn warnFunc, putSymbol putSymbolFunc) (Resolver, error) {
	r := mavenResolver{
		lang:      lang,
		name:      mavenWorkspaceName,
		warn:      warn,
		data:      NewStringMultiSet(),
		artifacts: make(map[string]label.Label),
	}

	log.Println("loading configuration from:", installFile)

	// Try v2 format first
	v2, err := loadLockfileV2(installFile)
	if err == nil && v2.Version == "2" {
		return newResolverFromV2(&r, v2, mavenWorkspaceName, putSymbol)
	}

	// Fallback to v1 format
	c, err := loadConfiguration(installFile)
	if err != nil {
		return nil, fmt.Errorf("loading configuration %s: %w", installFile, err)
	}

	return newResolverFromV1(&r, c, mavenWorkspaceName, putSymbol)
}

func newResolverFromV1(r *mavenResolver, c *configFile, mavenWorkspaceName string, putSymbol putSymbolFunc) (Resolver, error) {
	for _, dep := range c.DependencyTree.Dependencies {
		log.Println("loading v1 dep:", dep.Coord)
		coord, err := ParseCoordinate(dep.Coord)
		if err != nil {
			return nil, fmt.Errorf("failed to parse coordinate %v: %w", dep.Coord, err)
		}
		from := label.Label{Repo: mavenWorkspaceName, Name: bazel.CleanupLabelName(coord.ArtifactString())}
		labelString := from.String()
		r.artifacts[dep.Coord] = from

		for _, pkg := range dep.Packages {
			r.data.Add(pkg, labelString)
			putSymbol(resolver.NewSymbol(sppb.ImportType_PACKAGE, pkg, mavenWorkspaceName, from))
		}
	}

	return r, nil
}

func newResolverFromV2(r *mavenResolver, lf *LockfileV2, mavenWorkspaceName string, putSymbol putSymbolFunc) (Resolver, error) {
	// Sort artifact IDs for deterministic iteration
	ids := make([]string, 0, len(lf.Packages))
	for id := range lf.Packages {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		packages := lf.Packages[id]
		from := label.Label{Repo: mavenWorkspaceName, Name: bazel.CleanupLabelName(id)}
		labelString := from.String()
		r.artifacts[id] = from

		for _, pkg := range packages {
			r.data.Add(pkg, labelString)
			putSymbol(resolver.NewSymbol(sppb.ImportType_PACKAGE, pkg, mavenWorkspaceName, from))
		}
	}

	return r, nil
}

func (r *mavenResolver) Name() string {
	return r.name
}

func (r *mavenResolver) Resolve(pkg string) (label.Label, error) {
	v, found := r.data.Get(pkg)
	if !found {
		return label.NoLabel, fmt.Errorf("package not found: %s", pkg)
	}

	switch len(v) {
	case 0:
		return label.NoLabel, errors.New("no external imports")

	case 1:
		var ret string
		for r := range v {
			ret = r
			break
		}
		return label.Parse(ret)

	default:
		r.warn("Java package %q is provided by more than one label.  Append one of the following to BUILD.bazel:", pkg)
		for k := range v {
			r.warn("# gazelle:resolve %s %s %s", r.lang, pkg, k)
		}

		return label.NoLabel, errors.New("many possible imports")
	}
}
