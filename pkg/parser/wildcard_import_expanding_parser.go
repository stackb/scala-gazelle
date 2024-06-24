package parser

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const debugWildcardImportExpandingParser = false

// WildcardImportExpandingParser is a Parser frontend that removes wildcard imports.
type WildcardImportExpandingParser struct {
	next  Parser
	rules map[label.Label]*sppb.Rule
}

func NewWildcardExpandingParser(next Parser) *WildcardImportExpandingParser {
	return &WildcardImportExpandingParser{
		next:  next,
		rules: make(map[label.Label]*sppb.Rule),
	}
}

// ParseScalaRule implements parser.Parser
func (p *WildcardImportExpandingParser) ParseScalaRule(c *config.Config, kind string, from label.Label, dir string, srcs ...string) (*sppb.Rule, error) {
	rule, err := p.next.ParseScalaRule(c, kind, from, dir, srcs...)
	if err != nil {
		return nil, err
	}

	for _, file := range rule.Files {
		for _, imp := range file.Imports {
			if _, ok := resolver.IsWildcardImport(imp); ok {
				log.Printf("%s: wildcard import: %s", file.Filename, imp)
			}
		}
	}

	return rule, nil
}

func (p *WildcardImportExpandingParser) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	return p.next.LoadScalaRule(from, rule)
}
