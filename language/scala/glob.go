package scala

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
)

func parseGlob(call *build.CallExpr) (glob rule.GlobValue) {
	for _, expr := range call.List {
		switch e := expr.(type) {
		case *build.AssignExpr:
			if ident, ok := e.LHS.(*build.Ident); ok {
				switch ident.Name {
				case "exclude":
					if list, ok := e.RHS.(*build.ListExpr); ok {
						for _, item := range list.List {
							switch elem := item.(type) {
							case *build.StringExpr:
								glob.Excludes = append(glob.Excludes, elem.Value)
							default:
								log.Printf("skipping glob list item expression: %+v (%T)", elem, elem)
							}
						}
					} else {
						log.Printf("skipping glob assign exclude (only list expressions are supported): %s = %T", ident.Name, e.RHS)
					}
				default:
					log.Printf("skipping glob assignment: %s (unrecognized property)", ident.Name)
				}
			}
		case *build.ListExpr:
			for _, item := range e.List {
				switch elem := item.(type) {
				case *build.StringExpr:
					glob.Patterns = append(glob.Patterns, elem.Value)
				default:
					log.Printf("skipping glob list item expression: %+v (%T)", elem, elem)
				}
			}
		default:
			log.Printf("skipping glob list expression: %T", e)
		}
	}

	return
}
