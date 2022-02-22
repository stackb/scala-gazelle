package scala

import (
	"io/fs"
	"log"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bmatcuk/doublestar/v4"
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
