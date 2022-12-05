package provider

import "github.com/bazelbuild/bazel-gazelle/label"

// mockImportProvider is a mock of the protoc.ImportProvider interface.
type mockImportProvider struct {
	imports map[label.Label][]string
}

func (p *mockImportProvider) Provided(lang, impLang string) map[label.Label][]string {
	return p.imports
}
