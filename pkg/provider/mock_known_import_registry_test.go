package provider

import "github.com/stackb/scala-gazelle/pkg/resolver"

// mockKnownImportRegistry is a mock of the resolver.KnownImportRegistry interface.
type mockKnownImportRegistry struct {
	// want supplies return values for the GetKnownImport function.
	want map[string]*resolver.KnownImport
	// got records calls from the PutKnownImport function.
	got []*resolver.KnownImport
	// putKnownImportErr supplies a return value for the PutKnownImport.
	putKnownImportErr error
}

func (r *mockKnownImportRegistry) GetKnownImport(imp string) (*resolver.KnownImport, bool) {
	if r.want == nil {
		return nil, false
	}
	got, ok := r.want[imp]
	return got, ok
}

func (r *mockKnownImportRegistry) PutKnownImport(known *resolver.KnownImport) error {
	r.got = append(r.got, known)
	return r.putKnownImportErr
}
