package scala

import (
	"errors"

	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func newScalaScope(scope resolver.Scope) (next resolver.Scope, err error) {
	// first attempt will be original scope
	scopes := []resolver.Scope{scope}

	// second attempt will try and match 'scala.*' symbols
	if scala, ok := scope.GetScope("scala"); ok {
		scopes = append(scopes, scala)
	} else {
		err = errors.New("scala symbols have not been provided")
	}

	// third attempt will try and match 'java.lang.*' symbols
	if java, ok := scope.GetScope("java.lang"); ok {
		scopes = append(scopes, java)
	} else {
		err = errors.New("java.lang symbols have not been provided")
	}

	// fourth attempt will try and remove '_root_.'
	scopes = append(scopes, resolver.NewTrimPrefixScope("_root_.", scope))

	next = resolver.NewChainScope(scopes...)
	return
}
