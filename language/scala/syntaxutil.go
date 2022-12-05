package scala

import (
	"fmt"
	"log"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
)

func stringList(list *build.ListExpr) (values []string) {
	for _, item := range list.List {
		switch elem := item.(type) {
		case *build.StringExpr:
			values = append(values, elem.Value)
		default:
			log.Printf("skipping glob list item expression: %+v (%T)", elem, elem)
		}
	}
	return
}

func globalStringList(file *rule.File, ident *build.Ident) ([]string, error) {
	value, err := resolveGlobalAssignment(file, ident.Name)
	if err != nil {
		return nil, fmt.Errorf("%s must resolve to a starlark List[String]: %v", ident.Name, err)
	}
	list, ok := value.(*build.ListExpr)
	if !ok {
		return nil, fmt.Errorf("%s must resolve to a starlark List[String]", ident.Name)
	}
	return stringList(list), nil
}

func resolveGlobalAssignment(file *rule.File, identName string) (build.Expr, error) {
	for _, stmt := range file.File.Stmt {
		switch t := stmt.(type) {
		case *build.AssignExpr:
			if ident, ok := t.LHS.(*build.Ident); ok {
				if ident.Name == identName {
					return t.RHS, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("unknown global identifier: %s", identName)
}
