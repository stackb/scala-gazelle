package mocks

import (
	"testing"

	resolver "github.com/stackb/scala-gazelle/pkg/resolver"
	mock "github.com/stretchr/testify/mock"
)

type PutSymbolsCapturer struct {
	Registry *Scope
	Got      []*resolver.Symbol
}

func (k *PutSymbolsCapturer) capture(known *resolver.Symbol) bool {
	k.Got = append(k.Got, known)
	return true
}

func NewSymbolsCapturer(t *testing.T) *PutSymbolsCapturer {
	c := &PutSymbolsCapturer{
		Registry: NewScope(t),
	}

	c.Registry.
		On("PutSymbol", mock.MatchedBy(c.capture)).
		Maybe().
		Return(nil)

	return c
}
