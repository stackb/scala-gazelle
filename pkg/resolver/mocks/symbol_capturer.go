package mocks

import (
	"testing"

	resolver "github.com/stackb/scala-gazelle/pkg/resolver"
	mock "github.com/stretchr/testify/mock"
)

type SymbolCapturer struct {
	Registry *Scope
	Got      []*resolver.Symbol
}

func (k *SymbolCapturer) capture(symbol *resolver.Symbol) bool {
	k.Got = append(k.Got, symbol)
	return true
}

func NewSymbolsCapturer(t *testing.T) *SymbolCapturer {
	c := &SymbolCapturer{
		Registry: NewScope(t),
	}

	c.Registry.
		On("PutSymbol", mock.MatchedBy(c.capture)).
		Maybe().
		Return(nil)

	return c
}
