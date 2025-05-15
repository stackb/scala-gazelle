package scala

import (
	"fmt"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// GetKnownScope implements part of the resolver.KnownScopeRegistry interface.
func (sl *scalaLang) GetKnownScope(name string) (resolver.Scope, bool) {
	scope, ok := sl.knownScopes[name]
	return scope, ok
}

// PutKnownScope implements part of the resolver.KnownScopeRegistry interface.
func (sl *scalaLang) PutKnownScope(name string, scope resolver.Scope) error {
	if _, ok := sl.knownScopes[name]; ok {
		return fmt.Errorf("duplicate known rule: %s", name)
	}
	sl.knownScopes[name] = scope
	return nil
}
