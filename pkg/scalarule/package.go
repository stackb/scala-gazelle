package scalarule

import (
	grule "github.com/bazelbuild/bazel-gazelle/rule"
)

// Package is responsible for instantiating a Rule interface for the given
// gazelle.Rule, parsing the attribute name given (typically 'srcs').
type Package interface {
	// ParseRule parses the given rule from the named attr (typically 'srcs').
	ParseRule(r *grule.Rule, attrName string) (Rule, error)
}
