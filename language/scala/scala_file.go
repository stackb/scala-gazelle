package scala

import (
	"log"
	"strings"

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

	imports []ScalaImport
}

func (l *scalaListener) addImportedType(importedType string) {
	l.imports = append(l.imports, ScalaImport{Name: importedType})
	log.Println("Added import", importedType)
}

func (l *scalaListener) addAllImportedType(importedType string) {
	// l.imports = append(l.imports, ScalaImport{Name: importedType})
	log.Println("Added all import", importedType)
}

func (l *scalaListener) EnterImport_(ctx *parser.Import_Context) {
	// log.Println("EnterImport_", ctx)
}

func (l *scalaListener) ExitImportExpr(ctx *parser.ImportExprContext) {
	// recog := ctx.GetParser()

	// log.Printf("ExitImport_ expr %T", t.StableId())
	stableID, ok := ctx.StableId().(*parser.StableIdContext)
	if !ok {
		return
	}

	log.Printf("ExitImport_ stableID %v (id=%+v)", stableID.GetText(), ctx.Id())
	log.Println("text: " + ctx.GetText())

	typeName := stableID.GetText()

	if isc, ok := ctx.ImportSelectors().(*parser.ImportSelectorsContext); ok {
		log.Println("isc text: " + isc.GetText())

		for i := 0; i < isc.GetChildCount(); i++ {
			child := isc.GetChild(i)
			switch childT := child.(type) {
			case *parser.ImportSelectorContext:
				log.Println("ImportSelectorContext text: " + childT.GetText())
				for _, tn := range childT.AllId() {
					log.Println("ImportSelectorContext id text: " + tn.GetText())
					l.addImportedType(typeName + "." + tn.GetText())
				}
			case *antlr.TerminalNodeImpl:
				if childT.GetText() == "_" {
					l.addAllImportedType(typeName)
				}
			}
			// if pc, ok := child.(antlr.ParseTree); ok {
			// 	log.Printf("isc child %d (%T): %v", i, child, pc.ToStringTree(recog.GetRuleNames(), recog))
			// } else {
			// 	log.Printf("isc child %d (%T): %+v", i, child, child)
			// }
		}

		// case of "import a.b.c.{D, E}"
		// for _, is := range isc.AllImportSelector() {
		// 	if s, ok := is.(*parser.ImportSelectorContext); ok {
		// 	}
		// }
	} else {
		if strings.HasSuffix(ctx.GetText(), "_") {
			l.addImportedType(ctx.GetText())
		} else {
			l.addImportedType(typeName)
		}
	}
}
