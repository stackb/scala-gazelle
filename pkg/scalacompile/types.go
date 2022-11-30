package scalacompile

// ScalaCompileSpec describes the symbols derived from attempting to compile a scala source file.
type ScalaCompileSpec struct {
	// NotFound is a list of types that were not found (e.g. "not found: value DateUtils")
	NotFound []*NotFoundSymbol `json:"notFound,omitempty"`
	// E.g. "object Session is not a member of package com.foo.core"
	NotMember []*NotMemberSymbol `json:"notMember,omitempty"`
}

type NotFoundSymbol struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type NotMemberSymbol struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Package string `json:"package"`
}
