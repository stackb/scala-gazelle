package autokeep

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	akpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/autokeep"
	scpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/cache"

	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

type DepsMap map[string]string

func MakeDeltaDeps(deps DepsMap, diagnostics *akpb.Diagnostics) *akpb.DeltaDeps {
	rules := make(map[string]*akpb.RuleDeps)
	delta := new(akpb.DeltaDeps)
	for _, e := range diagnostics.ScalacErrors {
		rule := rules[e.RuleLabel]
		if rule == nil {
			rule = new(akpb.RuleDeps)
			rule.Label = e.RuleLabel
			rule.BuildFile = e.BuildFile
		}
		switch t := e.Error.(type) {
		case *akpb.ScalacError_MissingSymbol:
			sym := t.MissingSymbol.Symbol
			if label, ok := deps[sym]; ok {
				log.Println("MATCH: ", sym, "is provided by", label)
				if len(rule.Deps) == 0 {
					delta.Add = append(delta.Add, rule)
				}
				rule.Deps = append(rule.Deps, label)
			} else {
				log.Printf("MISS (missing symbol not found): %q", sym)
			}
		case *akpb.ScalacError_NotAMemberOfPackage:
			sym := fmt.Sprintf("%s.%s", t.NotAMemberOfPackage.PackageName, t.NotAMemberOfPackage.Symbol)
			if label, ok := deps[sym]; ok {
				log.Println("MATCH: ", sym, "is provided by", label)
				if len(rule.Deps) == 0 {
					delta.Add = append(delta.Add, rule)
				}
				rule.Deps = append(rule.Deps, label)
			} else {
				log.Printf("MISS (not found): %q (%d)", sym, len(deps))
			}
		case *akpb.ScalacError_BuildozerUnusedDep:
			delta.Remove = append(delta.Remove, rule)
			rule.Deps = append(rule.Deps, t.BuildozerUnusedDep.UnusedDep)
			rule = nil
		}
	}
	return delta
}

func MergeDepsFromCacheFile(deps DepsMap, filename string) error {
	cache := &scpb.Cache{}
	if err := protobuf.ReadFile(filename, cache); err != nil {
		return err
	}
	MergeDepsFromCache(deps, cache)
	return nil
}

func MergeDepsFromCache(deps DepsMap, cache *scpb.Cache) {
	// mapping of symbol fqn -> source rule label
	for _, rule := range cache.Rules {
		for _, file := range rule.Files {
			for _, s := range file.Classes {
				deps[s] = rule.Label
			}
			for _, s := range file.Objects {
				deps[s] = rule.Label
			}
			for _, s := range file.Traits {
				deps[s] = rule.Label
			}
			for _, s := range file.Types {
				deps[s] = rule.Label
			}
		}
	}
}

func MergeDepsFromImportsFile(deps DepsMap, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return MergeDepsFromImports(deps, f)
}

func MergeDepsFromImports(deps DepsMap, in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			log.Println("WARN: bad line:", line)
			continue
		}
		imp := fields[0]
		label := fields[1]
		deps[imp] = label
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func ApplyDeltaDeps(delta *akpb.DeltaDeps, wantKeepComment bool) error {
	for _, ruleDeps := range delta.Add {
		if err := add(ruleDeps, wantKeepComment); err != nil {
			return err
		}
	}
	for _, ruleDeps := range delta.Remove {
		if err := remove(ruleDeps); err != nil {
			return err
		}
	}
	return nil
}

func add(ruleDeps *akpb.RuleDeps, wantKeepComment bool) error {
	lbl, err := label.Parse(ruleDeps.Label)
	if err != nil {
		return err
	}
	file, err := rule.LoadFile(ruleDeps.BuildFile, lbl.Pkg)
	if err != nil {
		return err
	}
	r, err := findRuleInFile(file, lbl.Name)
	if err != nil {
		return err
	}
	depsExpr := r.Attr("deps")
	if depsExpr == nil {
		return fmt.Errorf("%v: 'deps' attribute not found", lbl)
	}
	depsList, ok := depsExpr.(*build.ListExpr)
	if !ok {
		return fmt.Errorf("%v: 'deps' attribute is not a list (%T)", lbl, depsExpr)
	}
	for _, dep := range ruleDeps.Deps {
		log.Printf("%v: adding dep %q", lbl, dep)
		depExpr := &build.StringExpr{Value: dep}
		if wantKeepComment {
			depExpr.Comments.Suffix = append(depExpr.Comments.Suffix, build.Comment{Token: "  # keep"})
		}
		depsList.List = append(depsList.List, depExpr)
	}
	if err := os.WriteFile(ruleDeps.BuildFile, file.Format(), 0o666); err != nil {
		return err
	}
	return nil
}

func remove(ruleDeps *akpb.RuleDeps) error {
	lbl, err := label.Parse(ruleDeps.Label)
	if err != nil {
		return err
	}
	file, err := rule.LoadFile(ruleDeps.BuildFile, lbl.Pkg)
	if err != nil {
		return err
	}
	r, err := findRuleInFile(file, lbl.Name)
	if err != nil {
		return err
	}
	depsExpr := r.Attr("deps")
	if depsExpr == nil {
		return fmt.Errorf("%v: 'deps' attribute not found", lbl)
	}
	depsList, ok := depsExpr.(*build.ListExpr)
	if !ok {
		return fmt.Errorf("%v: 'deps' attribute is not a list (%T)", lbl, depsExpr)
	}

	newList := &build.ListExpr{}
	for _, item := range depsList.List {
		keep := true
		if str, ok := item.(*build.StringExpr); ok {
			for _, dep := range ruleDeps.Deps {
				if str.Value == dep {
					keep = false
					break
				}
			}
		}
		if keep {
			newList.List = append(newList.List, item)
		}
	}
	r.SetAttr("deps", newList)

	if err := os.WriteFile(ruleDeps.BuildFile, file.Format(), 0o666); err != nil {
		return err
	}
	return nil
}

func findRuleInFile(file *rule.File, name string) (*rule.Rule, error) {
	for _, r := range file.Rules {
		if r.Name() == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("%s: rule not found: %s", file.Path, name)
}
