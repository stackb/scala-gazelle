package scala

import (
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func shouldKeepDepExpr(sc *scalaConfig, expr build.Expr) bool {
	// does it have a '# keep' directive?
	if rule.ShouldKeep(expr) {
		return true
	}

	// is the expression something we can parse as a label?
	// If not, just leave it be.
	from := labelFromDepExpr(expr)
	if from == label.NoLabel {
		return true
	}

	// if we can find a provider for this label, remove it (it should get
	// resolved again)
	if sc.canProvide(from) {
		return false
	}

	// we didn't find an owner so keep just it, it's not a managed dependency.
	return true
}

// labelFromDepExpr returns the label from an expression like
// "@maven//:guava" or scala_dep("@maven//:guava")
func labelFromDepExpr(expr build.Expr) label.Label {
	switch t := expr.(type) {
	case *build.StringExpr:
		if from, err := label.Parse(t.Value); err != nil {
			return label.NoLabel
		} else {
			return from
		}
	case *build.CallExpr:
		if ident, ok := t.X.(*build.Ident); ok && ident.Name == "scala_dep" {
			if len(t.List) == 0 {
				return label.NoLabel
			}
			first := t.List[0]
			if str, ok := first.(*build.StringExpr); ok {
				if from, err := label.Parse(str.Value); err != nil {
					return label.NoLabel
				} else {
					return from
				}
			}
		}
	}

	return label.NoLabel
}

func buildKeepDepsList(sc *scalaConfig, current build.Expr) *build.ListExpr {
	deps := &build.ListExpr{}
	if current != nil {
		if listExpr, ok := current.(*build.ListExpr); ok {
			for _, expr := range listExpr.List {
				if shouldKeepDepExpr(sc, expr) {
					deps.List = append(deps.List, expr)
				}
			}
		}
	}
	return deps
}

func addResolvedDeps(deps *build.ListExpr, sc *scalaConfig, kind string, from label.Label, imports resolver.ImportMap) {
	// make a mapping of final deps to be included.  Getting strange behavior by
	// just creating a build.ListExpr and sorting that list directly.
	kept := make(map[string]resolver.ImportMap)

	seen := make(map[label.Label]bool)
	seen[from] = true // self-label

	if from.Repo == "" {
		from.Repo = sc.config.RepoName
	}

	for _, imp := range imports {
		if imp.Known == nil || imp.Error != nil {
			continue
		}
		// relativize the dependency label.  For self-imports, this transforms into the empty label.
		dep := imp.Known.Label.Rel(from.Repo, from.Pkg)
		log.Println("addResolvedDeps dep:", dep)
		if seen[dep] {
			log.Println("addResolvedDeps seen!", dep)
			continue
		}
		if dep == label.NoLabel {
			log.Println("addResolvedDeps dep==label.NoLabel!", dep)
			continue
		}
		if dep == from {
			log.Println("addResolvedDeps dep==from!", dep)
			continue
		}
		if from.Equal(dep) {
			log.Println("addResolvedDeps from.Equal!", dep)
			continue
		}
		if isSameImport(sc, kind, from, dep) {
			log.Println("addResolvedDeps isSameImport!", dep, from)
			continue
		}

		kept[dep.String()] = imports
		log.Println("addResolvedDeps kept:", dep)
		seen[dep] = true
	}

	deps.List = append(deps.List, makeAnnotatedDepExprs(kept, sc.explainDeps)...)
}

func makeAnnotatedDepExprs(deps map[string]resolver.ImportMap, annotate bool) (exprs []build.Expr) {
	keys := make([]string, 0, len(deps))
	for key := range deps {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, dep := range keys {
		imports := deps[dep]
		str := &build.StringExpr{Value: dep}
		if annotate {
			imports.Annotate(&str.Comments, func(imp *resolver.Import) bool {
				return imp.Error == nil && imp.Known != nil
			})
		}
		exprs = append(exprs, str)
	}

	return
}

// isSameImport returns true if the "from" and "to" labels are the same,
// normalizing to the config.RepoName and performing label name remapping if the
// kind matches.
func isSameImport(sc *scalaConfig, kind string, from, to label.Label) bool {
	if from.Repo == "" {
		from = label.New(sc.config.RepoName, from.Pkg, from.Name)
	}
	if to.Repo == "" {
		to = label.New(sc.config.RepoName, to.Pkg, to.Name)
	}
	if mapping, ok := sc.labelNameRewrites[kind]; ok {
		from = mapping.Rewrite(from)
	}
	return from == to
}
