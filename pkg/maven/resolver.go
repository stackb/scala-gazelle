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
	ResolveWithDirectDependencies(pkg string) (label.Label, []label.Label, error)
}

// resolver finds Maven provided packages by reading the maven_install.json
// file from rules_jvm_external.
type resolver struct {
	name               string
	data               *StringMultiSet
	artifacts          map[string]label.Label
	directDependencies map[label.Label][]label.Label
}

func NewResolver(installFile, mavenWorkspaceName string) (Resolver, error) {
	r := resolver{
		name:               mavenWorkspaceName,
		data:               NewStringMultiSet(),
		artifacts:          make(map[string]label.Label),
		directDependencies: make(map[label.Label][]label.Label),
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

		for _, directCoord := range dep.DirectDependencies {
			dc, err := ParseCoordinate(directCoord)
			if err != nil {
				return nil, fmt.Errorf("failed to parse coordinate %v: %w", directCoord, err)
			}
			dl := label.New(mavenWorkspaceName, "", bazel.CleanupLabel(dc.ArtifactString()))
			r.directDependencies[l] = append(r.directDependencies[l], dl)
		}
		for _, pkg := range dep.Packages {
			r.data.Add(pkg, labelString)
			// log.Printf("maven: %v -> %v", pkg, l.String())
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

func (r *resolver) ResolveWithDirectDependencies(pkg string) (label.Label, []label.Label, error) {
	v, found := r.data.Get(pkg)
	if !found {
		return label.NoLabel, nil, fmt.Errorf("package not found: %s", pkg)
	}

	switch len(v) {
	case 0:
		return label.NoLabel, nil, errors.New("no external imports")

	case 1:
		var ret string
		for r := range v {
			ret = r
			break
		}
		from, err := label.Parse(ret)
		if err != nil {
			return label.NoLabel, nil, err
		}
		directs := r.directDependencies[from]
		return from, directs, nil

	default:
		log.Println("Append one of the following to BUILD.bazel:")
		for k := range v {
			log.Printf("# gazelle:resolve java %s %s", pkg, k)
		}

		return label.NoLabel, nil, errors.New("many possible imports")
	}
}
