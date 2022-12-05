package provider

import "github.com/stackb/scala-gazelle/pkg/resolver"

// mockKnownImportRegistry is a mock of the resolver.KnownImportRegistry interface.
type mockKnownImportRegistry struct {
	get    map[string]*resolver.KnownImport
	put    []*resolver.KnownImport
	putErr error
}

func (r *mockKnownImportRegistry) GetKnownImport(imp string) (*resolver.KnownImport, bool) {
	if r.get == nil {
		return nil, false
	}
	got, ok := r.get[imp]
	return got, ok
}

func (r *mockKnownImportRegistry) PutKnownImport(known *resolver.KnownImport) error {
	r.put = append(r.put, known)
	return r.putErr
}
