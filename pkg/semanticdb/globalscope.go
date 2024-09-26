package semanticdb

import (
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

var globalScope resolver.Scope

func SetGlobalScope(scope resolver.Scope) {
	globalScope = scope
}

func GetGlobalScope() resolver.Scope {
	return globalScope
}
