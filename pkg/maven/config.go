package maven

import (
	"encoding/json"
	"os"
)

type configFile struct {
	DependencyTree dependencyTree `json:"dependency_tree"`
}

func loadConfiguration(filename string) (*configFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var c configFile
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

type dependencyTree struct {
	ConflictResolution map[string]string `json:"conflict_resolution"`
	Dependencies       []Dep             `json:"dependencies"`
	Version            string            `json:"version"`
}

type Dep struct {
	Coord              string   `json:"coord"`
	Dependencies       []string `json:"dependencies"`
	DirectDependencies []string `json:"directDependencies"`
	File               string   `json:"file"`
	MirrorUrls         []string `json:"mirror_urls,omitempty"`
	Packages           []string `json:"packages"`
	Sha256             string   `json:"sha256,omitempty"`
	URL                string   `json:"url,omitempty"`
	Exclusions         []string `json:"exclusions,omitempty"`
}

// LockfileV2 represents the maven2_install.json structure
type LockfileV2 struct {
	InputArtifactsHash    int64                          `json:"__INPUT_ARTIFACTS_HASH"`
	ResolvedArtifactsHash int64                          `json:"__RESOLVED_ARTIFACTS_HASH"`
	Version               string                         `json:"version"`
	Artifacts             map[string]ArtifactV2          `json:"artifacts"`
	Dependencies          map[string][]string            `json:"dependencies"`
	Packages              map[string][]string            `json:"packages"`
	Repositories          map[string][]string            `json:"repositories"`
	Services              map[string]map[string][]string `json:"services"`
	ConflictResolution    map[string]string              `json:"conflict_resolution"`
	Skipped               []string                       `json:"skipped"`
}

// ArtifactV2 represents metadata for a single maven artifact
type ArtifactV2 struct {
	Version string            `json:"version"`
	Shasums map[string]string `json:"shasums"`
}

func loadLockfileV2(filename string) (*LockfileV2, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lf LockfileV2
	if err := json.NewDecoder(f).Decode(&lf); err != nil {
		return nil, err
	}

	return &lf, nil
}
