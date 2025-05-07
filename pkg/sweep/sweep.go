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

	// Flag to fixup the deps build building the target and parsing scalac output.
	//
	// gazelle:scala_fix_deps true
	ScalaFixDepsDirective = "scala_fix_deps"
)

const TransitiveCommentToken = "# TRANSITIVE"

// TransitiveAttr iterates through deps marked "TRANSITIVE" and removes them if the
// target still builds without it.
func TransitiveAttr(attrName string, file *rule.File, r *rule.Rule, from label.Label) (junk []string, err error) {
	expr := r.Attr(attrName)
	if expr == nil {
		return nil, nil
	}

	deps, isList := expr.(*build.ListExpr)
	if !isList {
		return nil, nil // some other condition we can't deal with
	}

	// check that the deps have at least one unknown dep
	var hasTransitiveDeps bool
	for _, expr := range deps.List {
		if str, ok := expr.(*build.StringExpr); ok {
			for _, suffix := range str.Comment().Suffix {
				if suffix.Token == TransitiveCommentToken {
					hasTransitiveDeps = true
					break
				}
			}
		}
	}
	if !hasTransitiveDeps {
		return nil, nil // nothing to do
	}

	// target should build first time, otherwise we can't check accurately.
	log.Println("🧱 transitive sweep:", from)

	if out, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode != 0 {
		log.Fatalf("sweep failed (must build cleanly on first attempt): %s", string(out))
	}

	// iterate the list backwards
	for i := len(deps.List) - 1; i >= 0; i-- {
		expr := deps.List[i]

		// look for transitive string dep expressions
		dep, ok := expr.(*build.StringExpr)
		if !ok {
			continue
		}
		var isTransitiveDep bool
		for _, suffix := range dep.Comment().Suffix {
			if suffix.Token == TransitiveCommentToken {
				isTransitiveDep = true
				break
			}
		}
		if !isTransitiveDep {
			continue
		}

		// reference of original list in case it does not build
		original := deps.List
		// reset deps with this one spliced out
		deps.List = collections.SliceRemoveIndex(deps.List, i)
		// save file to reflect change
		if err := file.Save(file.Path); err != nil {
			return nil, err
		}
		// see if it still builds
		if _, exitCode, _ := bazel.ExecCommand("bazel", "build", from.String()); exitCode == 0 {
			log.Println("- 💩 junk:", dep.Value)
			junk = append(junk, dep.Value)
		} else {
			log.Println("- 👑 keep:", dep.Value)
			deps.List = original
		}
	}

	// final save with possible last change
	if err := file.Save(file.Path); err != nil {
		return nil, err
	}

	return
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

func MakeTransitiveComment() build.Comment {
	return build.Comment{Token: TransitiveCommentToken}
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

func HasTransitiveRuleComment(r *rule.Rule) bool {
	for _, before := range r.Comments() {
		if before == TransitiveCommentToken {
			return true
		}
	}
	return false
}
