package maven

import (
	"errors"
	"fmt"
	"log"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/stackb/scala-gazelle/pkg/bazel"
)

type Resolver interface {
	Name() string
	Resolve(pkg string) (label.Label, error)
}

// resolver finds Maven provided packages by reading the maven_install.json
// file from rules_jvm_external.
type resolver struct {
	name string
	data *StringMultiSet
	// logger zerolog.Logger
}

func NewResolver(installFile, mavenWorkspaceName string) (Resolver, error) {
	r := resolver{
		name: mavenWorkspaceName,
		data: NewStringMultiSet(),
	}

	c, err := loadConfiguration(installFile)
	if err != nil {
		return nil, fmt.Errorf("loading configuration %s: %w", installFile, err)
	}

	for _, dep := range c.DependencyTree.Dependencies {
		for _, pkg := range dep.Packages {
			c, err := ParseCoordinate(dep.Coord)
			if err != nil {
				return nil, fmt.Errorf("failed to parse coordinate %v: %w", dep.Coord, err)
			}
			l := label.New(mavenWorkspaceName, "", bazel.CleanupLabel(c.ArtifactString()))
			r.data.Add(pkg, l.String())
			log.Printf("maven: %v -> %v", pkg, l.String())
		}
	}

	return &r, nil
}

func (r *resolver) Name() string {
	return r.name
}

func (r *resolver) Resolve(pkg string) (label.Label, error) {
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
		log.Println("Append one of the following to BUILD.bazel:")
		for k := range v {
			log.Printf("# gazelle:resolve java %s %s", pkg, k)
		}

		return label.NoLabel, errors.New("many possible imports")
	}
}

func LabelFromArtifact(artifact string) string {
	return label.New("maven", "", bazel.CleanupLabel(artifact)).String()
}
