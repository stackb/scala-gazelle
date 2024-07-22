package parser

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
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

	sc := scalaconfig.Get(c)

	if sc.ShouldAnnotateWildcardImports() {
		for _, file := range rule.Files {
			for _, imp := range file.Imports {
				if _, ok := resolver.IsWildcardImport(imp); ok {
					log.Printf("❗❗❗ %s: wildcard import: %s", file.Filename, imp)
				}
			}
		}
	}

	return rule, nil
}

func (p *WildcardImportExpandingParser) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	return p.next.LoadScalaRule(from, rule)
}

// if r.ctx.scalaConfig.ShouldAnnotateWildcardImports() && item.sym.Type == sppb.ImportType_PROTO_PACKAGE {
// 	if scope, ok := r.ctx.scope.GetScope(item.imp.Imp); ok {
// 		wildcardImport := item.imp.Src // original symbol name having underscore suffix
// 		r.handleWildcardImport(item.imp.Source, wildcardImport, scope)
// 	}
// }

// func (r *scalaRule) handleWildcardImport(file *sppb.File, imp string, scope resolver.Scope) {
// 	names := make([]string, 0)
// 	for _, name := range file.Names {
// 		if _, ok := scope.GetSymbol(name); ok {
// 			names = append(names, name)
// 		}
// 	}
// 	if len(names) > 0 {
// 		sort.Strings(names)
// 		log.Printf("[%s]: import %s.{%s}", file.Filename, strings.TrimSuffix(imp, "._"), strings.Join(names, ", "))
// 	}
// }
