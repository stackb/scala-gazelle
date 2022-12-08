package mocks

import (
	"testing"

	resolver "github.com/stackb/scala-gazelle/pkg/resolver"
	mock "github.com/stretchr/testify/mock"
)

type PutKnownImportsCapturer struct {
	Registry *KnownImportRegistry
	Got      []*resolver.KnownImport
}

func (k *PutKnownImportsCapturer) capture(known *resolver.KnownImport) bool {
	k.Got = append(k.Got, known)
	return true
}

func NewKnownImportsCapturer(t *testing.T) *PutKnownImportsCapturer {
	c := &PutKnownImportsCapturer{
		Registry: NewKnownImportRegistry(t),
	}

	c.Registry.
		On("PutKnownImport", mock.MatchedBy(c.capture)).
		Maybe().
		Return(nil)

	return c
}
