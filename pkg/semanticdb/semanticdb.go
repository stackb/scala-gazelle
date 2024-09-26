package semanticdb

import (
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

func SemanticImports(in *spb.TextDocument) []string {
	visitor := NewTextDocumentVisitor()
	visitor.VisitTextDocument(in)
	return visitor.SemanticImports()
}
