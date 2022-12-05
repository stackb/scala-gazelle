package maven

import (
	"errors"
	"fmt"

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
type putKnownImportFunc func(*resolver.KnownImport) error

func NewResolver(installFile, mavenWorkspaceName, lang string, warn warnFunc, putKnownImport putKnownImportFunc) (Resolver, error) {
	r := mavenResolver{
		lang:      lang,
		name:      mavenWorkspaceName,
		warn:      warn,
		data:      NewStringMultiSet(),
		artifacts: make(map[string]label.Label),
	}

	c, err := loadConfiguration(installFile)
	if err != nil {
		return nil, fmt.Errorf("loading configuration %s: %w", installFile, err)
	}

	for _, dep := range c.DependencyTree.Dependencies {
		c, err := ParseCoordinate(dep.Coord)
		if err != nil {
			return nil, fmt.Errorf("failed to parse coordinate %v: %w", dep.Coord, err)
		}
		l := label.New(mavenWorkspaceName, "", bazel.CleanupLabel(c.ArtifactString()))
		labelString := l.String()
		r.artifacts[dep.Coord] = l

		for _, pkg := range dep.Packages {
			r.data.Add(pkg, labelString)
			// log.Printf("maven: %v -> %v", pkg, l.String())
			putKnownImport(&resolver.KnownImport{
				Type:   sppb.ImportType_PACKAGE,
				Import: pkg,
				Label:  l,
			})
		}
	}

	return &r, nil
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
