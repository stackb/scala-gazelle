package scala

import (
	// "log"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/stackb/scala-gazelle/antlr/parser"
)

type ScalaFile struct {
	Name    string
	Imports []ScalaImport
}

func ParseScalaFile(filename string) (*ScalaFile, error) {
	is, err := antlr.NewFileStream(filename)
	if err != nil {
		return nil, err
	}
	lexer := parser.NewScalaLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewScalaParser(stream)
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true
	tree := p.CompilationUnit()

	listener := &scalaListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return &ScalaFile{
		Name:    filename,
		Imports: listener.imports,
	}, nil
}

type ScalaImport struct {
	Name string
}

type scalaListener struct {
	*parser.BaseScalaListener

	// temporary list of ImportExprs.  List is re-initialized when we enter an
	// import.
	importExprs []*parser.ImportExprContext

	imports []ScalaImport
}

func (l *scalaListener) EnterImport_(ctx *parser.Import_Context) {
	// log.Println("EnterImport_", ctx)
	l.importExprs = make([]*parser.ImportExprContext, 0)
}

func (l *scalaListener) EnterImportExpr(ctx *parser.ImportExprContext) {
	l.importExprs = append(l.importExprs, ctx)
}

func (l *scalaListener) ExitImport_(ctx *parser.Import_Context) {
	// log.Println("ExitImport_", len(l.importExprs))

	for _, expr := range ctx.AllImportExpr() {
		if t, ok := expr.(*parser.ImportExprContext); ok {
			// log.Printf("ExitImport_ expr %T", t.StableId())
			if s, ok := t.StableId().(*parser.StableIdContext); ok {
				// log.Printf("ExitImport_ stableId %v", s.GetText())
				l.imports = append(l.imports, ScalaImport{Name: s.GetText()})
			}
		}
	}
}
