package glob

import (
	"fmt"
	"os"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
)

type collector struct {
	file *rule.File
	dir  string
	srcs []string
}

func CollectFilenames(file *rule.File, dir string, expr build.Expr) ([]string, error) {
	c := collector{file: file, dir: dir}
	if err := c.fromExpr(expr); err != nil {
		return nil, err
	}
	return c.srcs, nil
}

// collectSourceFilesFromExpr returns a list of source files for the srcs
// attribute.  Each value is a repo-relative path.
func (c *collector) fromExpr(expr build.Expr) (err error) {
	switch t := expr.(type) {
	case *build.StringExpr:
		c.srcs = append(c.srcs, t.Value)
	case *build.BinaryExpr:
		c.fromExpr(t.X)
		c.fromExpr(t.Y)
	case *build.ListExpr:
		// example: ["foo.scala", "bar.scala"]
		for _, item := range t.List {
			c.fromExpr(item)
		}
	case *build.CallExpr:
		// example: glob(["**/*.scala"])
		if ident, ok := t.X.(*build.Ident); ok {
			switch ident.Name {
			case "glob":
				g := Parse(c.file, t)
				c.srcs = append(c.srcs, Apply(g, os.DirFS(c.dir))...)
			default:
				err = fmt.Errorf("not attempting to resolve function call %v(): consider making this simpler", ident.Name)
			}
		} else {
			err = fmt.Errorf("not attempting to resolve call expression %+v: consider making this simpler", t)
		}
	case *build.Ident:
		// example: srcs = LIST_OF_SOURCES
		var srcs []string
		srcs, err = globalStringList(c.file, t)
		if err != nil {
			err = fmt.Errorf("failed to resolve resolve identifier %q (consider inlining it): %w", t.Name, err)
		}
		c.srcs = append(c.srcs, srcs...)
	case nil:
		break
	default:
		err = fmt.Errorf("uninterpretable 'srcs' attribute type: %T", t)
	}

	return
}
