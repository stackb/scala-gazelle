package sweep

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/scala-gazelle/pkg/bazel"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

const (
	// Turn on the dep sweeper
	//
	// gazelle:scala_sweep_transitive_deps true
	ScalaSweepTransitiveDepsDirective = "scala_sweep_transitive_deps"

	// Flag to preserve deps if the label is not known to be needed from the
	// imports (legacy migration mode).
	//
	// gazelle:scala_keep_unknown_deps true
	ScalaKeepUnknownDepsDirective = "scala_keep_unknown_deps"
)

const TransitiveCommentToken = "# TRANSITIVE"

// TransitiveAttr iterates through deps marked "TRANSITIVE" and removes them if
// the target still builds without it.
func TransitiveAttr(attrName string, r *rule.Rule, file *rule.File, from label.Label) error {
	expr := r.Attr(attrName)
	if expr == nil {
		return nil
	}

	deps, isList := expr.(*build.ListExpr)
	if !isList {
		return nil // some other condition we can't deal with
	}

	// target should build first time, otherwise we can't check accurately.
	log.Println("🧱 transitive sweep:", from)

	if out, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode != 0 {
		log.Fatalf("sweep failed (must build cleanly on first attempt): %s", string(out))
	}

	for i := len(deps.List) - 1; i >= 0; i-- {
		expr := deps.List[i]
		switch t := expr.(type) {
		case *build.StringExpr:
			if len(t.Comments.Suffix) != 1 {
				continue
			}
			if t.Comments.Suffix[0].Token != "# TRANSITIVE" {
				continue
			}

			dep, err := label.Parse(t.Value)
			if err != nil {
				return err
			}
			deps.List = collections.SliceRemoveIndex(deps.List, i)

			if err := file.Save(file.Path); err != nil {
				return err
			}

			if _, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode == 0 {
				log.Println("- 💩 junk:", dep)
			} else {
				log.Println("- 👑 keep:", dep)
				deps.List = collections.SliceInsertAt(deps.List, i, expr)
			}
		}

	}

	if err := file.Save(file.Path); err != nil {
		return err
	}

	return nil
}

func RemoveSweepDirective(file *rule.File) error {
	if file == nil {
		return nil
	}
	// if this file has the sweep directive, remove it
	for _, d := range file.Directives {
		if d.Key == ScalaSweepTransitiveDepsDirective && d.Value == "true" {
			old := []byte(fmt.Sprintf("# gazelle:%s true\n", ScalaSweepTransitiveDepsDirective))
			new := []byte{'\n'}
			file.Content = bytes.Replace(file.Content, old, new, -1)
			// file.Sync()
			if err := file.Save(file.Path); err != nil {
				return err
			}
			// log.Panicln("saved it!", file.Path)
		}
	}
	return nil
}

func SetKeepDepsDirective(file *rule.File, value bool) error {
	if file == nil {
		return nil
	}
	// if this file has the sweep directive, remove it
	for _, d := range file.Directives {
		if d.Key == ScalaKeepUnknownDepsDirective {
			// log.Panicln("found it!", file.Path)
			old := []byte(fmt.Sprintf("# gazelle:%s %t", ScalaKeepUnknownDepsDirective, !value))
			new := []byte(fmt.Sprintf("# gazelle:%s %t", ScalaKeepUnknownDepsDirective, value))
			log.Println("OLD:", string(file.Content))
			file.Content = bytes.Replace(file.Content, old, new, -1)
			// file.Sync()
			log.Println("NEW:", string(file.Content))

			if err := file.Save(file.Path); err != nil {
				return err
			}
			stat, err := os.Stat(file.Path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(file.Path, file.Content, stat.Mode()); err != nil {
				return err
			}
			data, err := os.ReadFile(file.Path)
			if err != nil {
				return err
			}
			log.Panicln("saved it!", file.Path, "FILE DATA:\n", string(data))
		}
	}
	return nil
}

func MakeTransitiveDep(dep label.Label) *build.StringExpr {
	expr := &build.StringExpr{Value: dep.String()}
	expr.Comment().Suffix = append(expr.Comment().Suffix, build.Comment{Token: TransitiveCommentToken})
	return expr
}

// IsTransitiveDep returns whether e is marked with a "# TRANSITIVE" comment.
func IsTransitiveDep(e build.Expr) bool {
	for _, c := range e.Comment().Suffix {
		text := strings.TrimSpace(c.Token)
		if text == TransitiveCommentToken {
			return true
		}
	}
	return false
}
