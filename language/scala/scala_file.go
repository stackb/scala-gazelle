package scala

import (
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/stackb/scala-gazelle/antlr/parser"
)

type ScalaFile struct {
	Name string
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

	antlr.ParseTreeWalkerDefault.Walk(&scalaListener{}, tree)

	return &ScalaFile{
		Name: filename,
	}, nil
}

type scalaListener struct {
	*parser.BaseScalaListener

	stack []int
}
