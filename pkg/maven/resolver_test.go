package maven

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func TestNewResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		content    string
		wantLabels []string
	}{
		"v1 format": {
			content: `{
				"dependency_tree": {
					"dependencies": [
						{
							"coord": "xml-apis:xml-apis:1.4.01",
							"dependencies": [],
							"directDependencies": [],
							"file": "xml-apis-1.4.01.jar",
							"packages": ["javax.xml", "javax.xml.parsers"],
							"sha256": "abc123"
						}
					],
					"version": "0.1.0"
				}
			}`,
			wantLabels: []string{
				"@maven//:xml_apis_xml_apis",
				"@maven//:xml_apis_xml_apis",
			},
		},
		"v2 format": {
			content: `{
				"__INPUT_ARTIFACTS_HASH": 123,
				"__RESOLVED_ARTIFACTS_HASH": 456,
				"version": "2",
				"artifacts": {
					"xml-apis:xml-apis": {
						"version": "1.4.01",
						"shasums": {"jar": "abc123"}
					}
				},
				"packages": {
					"xml-apis:xml-apis": ["javax.xml", "javax.xml.parsers"]
				},
				"dependencies": {},
				"repositories": {},
				"services": {},
				"conflict_resolution": {},
				"skipped": []
			}`,
			wantLabels: []string{
				"@maven//:xml_apis_xml_apis",
				"@maven//:xml_apis_xml_apis",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			installFile := filepath.Join(tmpDir, "maven_install.json")
			if err := os.WriteFile(installFile, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}

			var gotSymbols []*resolver.Symbol
			putSymbol := func(s *resolver.Symbol) error {
				gotSymbols = append(gotSymbols, s)
				return nil
			}

			r, err := NewResolver(installFile, "maven", "scala", func(format string, args ...interface{}) {}, putSymbol)
			if err != nil {
				t.Fatalf("NewResolver: %v", err)
			}

			if r.Name() != "maven" {
				t.Errorf("Name() = %q, want %q", r.Name(), "maven")
			}

			var gotLabels []string
			for _, s := range gotSymbols {
				gotLabels = append(gotLabels, s.Label.String())
			}

			if diff := cmp.Diff(tc.wantLabels, gotLabels); diff != "" {
				t.Errorf("labels (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResolverV2Deterministic(t *testing.T) {
	// Create a v2 lockfile with multiple artifacts
	content := `{
		"__INPUT_ARTIFACTS_HASH": 123,
		"__RESOLVED_ARTIFACTS_HASH": 456,
		"version": "2",
		"artifacts": {},
		"packages": {
			"com.example:charlie": ["com.example.charlie"],
			"com.example:alpha": ["com.example.alpha"],
			"com.example:bravo": ["com.example.bravo"]
		},
		"dependencies": {},
		"repositories": {},
		"services": {},
		"conflict_resolution": {},
		"skipped": []
	}`

	tmpDir := t.TempDir()
	installFile := filepath.Join(tmpDir, "maven_install.json")
	if err := os.WriteFile(installFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Run multiple times to verify deterministic ordering
	var firstOrder []string
	for i := 0; i < 10; i++ {
		var gotSymbols []*resolver.Symbol
		putSymbol := func(s *resolver.Symbol) error {
			gotSymbols = append(gotSymbols, s)
			return nil
		}

		_, err := NewResolver(installFile, "maven", "scala", func(format string, args ...interface{}) {}, putSymbol)
		if err != nil {
			t.Fatalf("NewResolver: %v", err)
		}

		var order []string
		for _, s := range gotSymbols {
			order = append(order, s.Name)
		}

		if i == 0 {
			firstOrder = order
			// Verify alphabetical ordering
			want := []string{"com.example.alpha", "com.example.bravo", "com.example.charlie"}
			if diff := cmp.Diff(want, order); diff != "" {
				t.Errorf("expected alphabetical order (-want +got):\n%s", diff)
			}
		} else {
			if diff := cmp.Diff(firstOrder, order); diff != "" {
				t.Errorf("iteration %d: order is not deterministic (-first +current):\n%s", i, diff)
			}
		}
	}
}

func TestResolverResolve(t *testing.T) {
	content := `{
		"__INPUT_ARTIFACTS_HASH": 123,
		"__RESOLVED_ARTIFACTS_HASH": 456,
		"version": "2",
		"artifacts": {},
		"packages": {
			"com.example:foo": ["com.example.foo", "com.example.foo.bar"]
		},
		"dependencies": {},
		"repositories": {},
		"services": {},
		"conflict_resolution": {},
		"skipped": []
	}`

	tmpDir := t.TempDir()
	installFile := filepath.Join(tmpDir, "maven_install.json")
	if err := os.WriteFile(installFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := NewResolver(installFile, "maven", "scala", func(format string, args ...interface{}) {}, func(s *resolver.Symbol) error { return nil })
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	for name, tc := range map[string]struct {
		pkg       string
		wantLabel label.Label
		wantErr   bool
	}{
		"found package": {
			pkg:       "com.example.foo",
			wantLabel: label.Label{Repo: "maven", Name: "com_example_foo"},
		},
		"found nested package": {
			pkg:       "com.example.foo.bar",
			wantLabel: label.Label{Repo: "maven", Name: "com_example_foo"},
		},
		"not found": {
			pkg:     "com.example.notfound",
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := r.Resolve(tc.pkg)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if diff := cmp.Diff(tc.wantLabel, got); diff != "" {
				t.Errorf("Resolve (-want +got):\n%s", diff)
			}
		})
	}
}
