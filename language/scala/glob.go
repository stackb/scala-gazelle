package scala

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/davecgh/go-spew/spew"
	"github.com/stackb/scala-gazelle/pkg/starlarkeval"
	"go.starlark.net/starlark"
)

func parseGlob(file *rule.File, call *build.CallExpr) (glob rule.GlobValue) {
	for i, expr := range call.List {
		switch e := expr.(type) {
		case *build.AssignExpr:
			if ident, ok := e.LHS.(*build.Ident); ok {
				switch ident.Name {
				case "exclude":
					switch rhs := e.RHS.(type) {
					case *build.ListExpr:
						glob.Excludes = append(glob.Excludes, stringList(rhs)...)
					case *build.Ident:
						values, err := globalStringList(file, rhs)
						if err != nil {
							log.Printf("skipping list expression elem: %v", err)
							break
						}
						glob.Excludes = append(glob.Excludes, values...)
					default:
						log.Printf("skipping glob assign exclude (only list expressions are supported): %s = %T", ident.Name, e.RHS)
					}
				default:
					log.Printf("skipping glob assignment: %s (unrecognized property)", ident.Name)
				}
			}
		case *build.ListExpr:
			glob.Patterns = append(glob.Patterns, stringList(e)...)
		case *build.Ident:
			values, err := globalStringList(file, e)
			if err != nil {
				log.Printf("skipping list expression elem: %v", err)
				break
			}
			glob.Patterns = append(glob.Patterns, values...)
		default:
			if false {
				spew.Dump(call)
				log.Printf("skipping glob list expression %d: %T in %+v", i, e, call)
			}
		}
	}

	return
}

func applyGlob(glob rule.GlobValue, fs fs.FS) (srcs []string) {
	// part 1: gather candidates
	includes := []string{}
	for _, pattern := range glob.Patterns {
		names, err := doublestar.Glob(fs, pattern)
		// log.Printf("names for pattern %s in %s: %v", pattern, dir, names)
		if err != nil {
			// doublestar.Match returns only one possible error, and
			// only if the pattern is not valid.
			log.Printf("error during doublestar.Glob: %v (pattern invalid: %v)", err, pattern)
			continue
		}
		includes = append(includes, names...)
	}

	// part 2: filter candidates
	if len(glob.Excludes) > 0 {
	loop:
		for _, name := range includes {
			for _, exclude := range glob.Excludes {
				if ok, _ := doublestar.PathMatch(exclude, name); ok {
					continue loop
				}
			}
			srcs = append(srcs, name)
		}
	} else {
		srcs = append(srcs, includes...)
	}

	return
}

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

func evalGlobalIdentifier(file *rule.File, identName string) (starlark.Value, error) {
	in := bytes.NewReader(file.Format())
	interpreter := starlarkeval.NewInterpreter(log.Printf)
	if err := interpreter.Exec(file.Path, in); err != nil {
		return nil, err
	}
	value := interpreter.GetGlobal(identName)
	if value == nil {
		return nil, fmt.Errorf("unknown global identifier: %s", identName)
	}
	return value, nil
}
