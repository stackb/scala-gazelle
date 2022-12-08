package scalarule

import grule "github.com/bazelbuild/bazel-gazelle/rule"

// Package is presented to the rule provider and is used to access the parser.
type Package interface {
	// ParseRule parses the given rule from the named attr (typically 'srcs').
	ParseRule(r *grule.Rule, attrName string) (Rule, error)
}
