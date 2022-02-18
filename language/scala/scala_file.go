package scala

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/stackb/scala-gazelle/antlr/parser"
)

const debugParseScalaFile = false

type ScalaFile struct {
	Name    string
	Imports []ScalaImport
}

func ParseScalaFile(dir, base string) (*ScalaFile, error) {
	then := time.Now()

	filename := filepath.Join(dir, base)

	log.Println("parsing", filename, "...")

	is, err := antlr.NewFileStream(filename)
	if err != nil {
		return nil, err
	}
	lexer := parser.NewScalaLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewScalaParser(stream)
	if debugParseScalaFile {
		p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	} else {
		p.RemoveErrorListeners()
	}
	p.BuildParseTrees = true
	tree := p.CompilationUnit()

	listener := &scalaListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	dt := time.Now().Sub(then)
	log.Println("parsed", filename, "in", dt)

	return &ScalaFile{
		Name:    base,
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
	if debugParseScalaFile {
		log.Println("Added import", importedType)
	}
}

func (l *scalaListener) addAllImportedType(importedType string) {
	l.imports = append(l.imports, ScalaImport{Name: importedType + "._"})
	if debugParseScalaFile {
		log.Println("Added all import", importedType+"._")
	}
}

func (l *scalaListener) ExitImportExpr(ctx *parser.ImportExprContext) {
	stableID, ok := ctx.StableId().(*parser.StableIdContext)
	if !ok {
		return
	}

	typeName := stableID.GetText()

	if isc, ok := ctx.ImportSelectors().(*parser.ImportSelectorsContext); ok {
		for i := 0; i < isc.GetChildCount(); i++ {
			child := isc.GetChild(i)
			switch childT := child.(type) {
			case *parser.ImportSelectorContext:
				for j := 0; j < childT.GetChildCount(); j++ {
					sel := childT.GetChild(j)
					switch selT := sel.(type) {
					case *antlr.TerminalNodeImpl:
						// ImportSelector has two forms: a.b.c.{D} or a.b.c.{D
						// => E}.  In both cases we only really care about the
						// first child.  Leaving the loop here for possible
						// future cases.
						if j == 0 {
							l.addImportedType(typeName + "." + selT.GetText())
						}
						// case antlr.ParseTree:
						// 	log.Printf("sel: %T %v", selT, selT.ToStringTree(ctx.GetParser().GetRuleNames(), ctx.GetParser()))
					}
				}
			case *antlr.TerminalNodeImpl:
				if childT.GetText() == "_" {
					// handles the case like "a.b.c.{D => E, _}"
					l.addAllImportedType(typeName)
				}
			}
		}
	} else {
		if strings.HasSuffix(ctx.GetText(), "_") {
			// handles the case like "a.b.c._"
			l.addAllImportedType(typeName)
		} else {
			// handles the case like "a.b.c.D"
			l.addImportedType(typeName)
		}
	}
}
