package resolver

import (
	"fmt"
)

// ChainKnownImportRegistry implements KnownImportRegistry over a chain of registries.
type ChainKnownImportRegistry struct {
	chain []KnownImportRegistry
}

func NewChainKnownImportRegistry(chain ...KnownImportRegistry) *ChainKnownImportRegistry {
	return &ChainKnownImportRegistry{
		chain: chain,
	}
}

// PutKnownImport is not supported and will panic.
func (r *ChainKnownImportRegistry) PutKnownImport(known *KnownImport) error {
	return fmt.Errorf("unsupported operation: PutKnownImport")
}

// GetKnownImport implements part of the KnownImportRegistry interface
func (r *ChainKnownImportRegistry) GetKnownImport(imp string) (*KnownImport, bool) {
	for _, next := range r.chain {
		if known, ok := next.GetKnownImport(imp); ok {
			return known, true
		}
	}
	return nil, false
}

// GetKnownImports implements part of the KnownImportRegistry interface
func (r *ChainKnownImportRegistry) GetKnownImports(prefix string) []*KnownImport {
	for _, next := range r.chain {
		if known := next.GetKnownImports(prefix); len(known) > 0 {
			return known
		}
	}
	return nil
}
