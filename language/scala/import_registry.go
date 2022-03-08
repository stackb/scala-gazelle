package scala

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// ScalaImportRegistry implementations are capable of disambiguating wildcard
// and other imports.
type ScalaImportRegistry interface {
	// Disambiguate takes an import token (typically a wildcard like
	// 'com.foo._') which has resolved to multiple labels (e.g. [//lib:a ->
	// com.foo.Bar, //lib:b -> com.foo.Baz], both of which provide types in
	// 'com.foo._'), and the from label representing the rule being resolved.
	// Presumably an implementation would look through the symbols defined in
	// the sources of 'from' and resolve which types in the wildcard were
	// actually referenced. It returns a narrower label list. If the result is
	// length 0, the import 'com.foo._' was determined to be an unused import.
	// If the result is length 1, this is the ideal case.  If the len(result) >
	// 1, all said labels should be included in deps.  If the result remains
	// ambiguous, error is returned, possibly with a non-empty list of labels
	// that represent best-effort results.
	Disambiguate(c *config.Config, imp string, labels []label.Label, from label.Label) ([]label.Label, error)
	// Get the label that provides the given import
	Provider(imp string) (label.Label, bool)
	// Completions returns concrete symbols under the given prefix
	Completions(prefix string) map[string]label.Label
}

func newImportRegistry(scalaRuleRegistry ScalaRuleRegistry, scalaCompiler ScalaCompiler) *importRegistry {
	return &importRegistry{
		importsOut:        "/tmp/scala-gazelle-imports.csv",
		scalaRuleRegistry: scalaRuleRegistry,
		scalaCompiler:     scalaCompiler,
		provides:          make(map[label.Label][]string),
		imports:           make(map[string]label.Label),
		classes:           make(map[string][]label.Label),
	}
}

// importRegistry implements ScalaImportRegistry.
type importRegistry struct {
	scalaRuleRegistry ScalaRuleRegistry
	scalaCompiler     ScalaCompiler
	// provides is a mapping of 'from' labels representing the concrete types that from provides.
	provides map[label.Label][]string
	// imports is a mapping of an import to the label that provides it, the
	// inverse map of provides.
	imports map[string]label.Label
	// classes is a mapping from the last symbol in a import to the labels that provides them.
	classes map[string][]label.Label
	// importsOut is the name of a file to write the index to
	importsOut string
}

func (ir *importRegistry) OnResolve() {
	// invert the provides map.
	for from, imports := range ir.provides {
		for _, imp := range imports {
			if _, ok := ir.imports[imp]; ok {
				if debug {
					log.Printf("importRegistry: duplicate provider of %q: %v %v", imp, ir.imports[imp], from)
				}
			}
			ir.imports[imp] = from
			class, ok := importClass(imp)
			if ok {
				ir.classes[class] = append(ir.classes[class], from)
			}
		}
	}

	if err := ir.writeImports(); err != nil {
		log.Fatalln("could not write imports file:", err)
	}
}

func (ir *importRegistry) Provider(imp string) (label.Label, bool) {
	from, ok := ir.imports[imp]
	return from, ok
}

func (ir *importRegistry) Provides(l label.Label, imports []string) {
	ir.provides[l] = append(ir.provides[l], imports...)
}

// Completions performs a prefix scan of the imports map and returns a map of types
// that are in that import.  For example, 'complete(java.util._) would return
// (Map -> //jdk:lang, ...)
func (ir *importRegistry) Completions(imp string) map[string]label.Label {
	completions := make(map[string]label.Label)

	// transform 'java.util._' to 'java.util.'
	prefix := strings.TrimSuffix(imp, "_")

	// log.Printf("completing %q", prefix)

	for i, from := range ir.imports {
		if strings.HasPrefix(i, prefix) {
			if i == prefix {
				continue
			}
			base := strings.TrimPrefix(i[len(prefix):], ".")
			completions[base] = from
		}
	}

	return completions
}

func (ir *importRegistry) Disambiguate(c *config.Config, imp string, labels []label.Label, from label.Label) ([]label.Label, error) {
	debug := false

	// step 0: check override specs
	if sc := getScalaConfig(c); sc != nil {
		for _, override := range sc.overrides {
			// log.Printf("%v: check match override %q with override: %v", from, imp, override.imp.Imp)
			if ok, _ := doublestar.Match(override.imp.Imp, imp); ok {
				if debug {
					log.Printf("%v: disambiguated %q with override: %v", from, imp, override.dep)
				}
				return []label.Label{override.dep}, nil
			}
		}
	}

	if debug {
		log.Printf("%v: disambiguating %q, candidate labels: %v", from, imp, labels)
	}

	// step 1: get completion symbols for the import.  For example, if we had
	// the import 'java.util._', get the set (java.util.List, java.util.Map,
	// ...)
	completions := ir.Completions(imp)
	if len(completions) == 0 {
		return nil, fmt.Errorf("no completions known for %v (aborting disambiguation attempt of %q)", from, imp)
	}

	if debug {
		for i, c := range completions {
			log.Printf("completion universe member %q: %v", i, c)
		}
	}

	// step 2: gather the list of srcs in 'from' and filter them such that
	// only those that explicitly use the import are retained.
	rule, ok := ir.scalaRuleRegistry.GetScalaRule(from)
	if !ok {
		return labels, fmt.Errorf("no srcs known for %v (aborting disambiguation attempt of %q)", from, imp)
	}

	files := []*index.ScalaFileSpec{}
	for _, file := range rule.Srcs {
		for _, i := range file.Imports {
			if imp == i {
				files = append(files, file)
				break
			}
		}
	}
	if len(files) == 0 {
		return labels, fmt.Errorf("rule %v did not actually list %[2]q as an import (aborting disambiguation attempt of %[2]q)", from, imp)
	}

	// step 3: use the scala compiler to list the unknown types in the source file.
	types := make(map[string]bool)

	for _, file := range files {
		compilation, err := ir.scalaCompiler.Compile(c.RepoRoot, file.Filename)
		if err != nil {
			return labels, fmt.Errorf("rule %v: error while disambiguating import %q in file %s: %w", from, imp, file.Filename, err)
		}
		for _, sym := range compilation.NotFound {
			types[sym.Name] = true
		}
		for _, sym := range compilation.NotMember {
			if sym.Package == imp {
				types[sym.Name] = true
			}
		}
		// augment the types map with any imports they specifically named in an import.
		for _, fileImp := range file.Imports {
			if strings.HasPrefix(fileImp, imp) {
				sym := strings.TrimPrefix(fileImp[len(imp):], ".")
				if sym != "" {
					types[sym] = true
				}
			}
		}
		// also all applied names
		for _, sym := range file.Names {
			types[sym] = true
		}
	}

	if debug {
		for i, c := range types {
			log.Printf("possible completion %q: %v", i, c)
		}
	}

	// step 4: process all the unknown types in srcs.  If we find a completion
	// match, we assume they are using this actual symbol.
	match := make(map[label.Label]bool)

	for sym := range types {
		from, ok := completions[sym]
		if ok {
			match[from] = true
		}
	}

	if len(match) == 0 {
		return labels, fmt.Errorf("no completion matches found for %v in %v (aborting disambiguation attempt of %q)", from, types, imp)
	}

	actuals := make([]label.Label, len(match))
	i := 0
	for from := range match {
		actuals[i] = from
		i++
	}

	return actuals, nil
}

func importClass(imp string) (string, bool) {
	idx := strings.LastIndex(imp, ".")
	if idx <= 0 || idx == len(imp) {
		return imp, false
	}
	return imp[idx+1:], true
}

// CrossResolve implements the CrossResolver interface.
func (ir *importRegistry) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	if lang != ScalaLangName {
		return nil
	}

	if from, ok := ir.imports[imp.Imp]; ok {
		return []resolve.FindResult{{Label: from}}
	}

	// class, _ := importClass(strings.TrimSuffix(imp.Imp, "._"))
	// if from, ok := ir.classes[class]; ok && len(from) == 1 {
	// 	// log.Println("success exact match class check:", class, from)
	// 	return []resolve.FindResult{{Label: from[0]}}
	// }

	return nil
}

func (r *importRegistry) writeImports() error {
	if r.importsOut == "" {
		return nil
	}

	f, err := os.Create(r.importsOut)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	keys := make([]string, len(r.imports))
	i := 0
	for k := range r.imports {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for _, imp := range keys {
		w.WriteString(imp)
		w.WriteString(",")
		w.WriteString(r.imports[imp].String())
		w.WriteString("\n")
	}

	w.Flush()

	log.Println("Wrote", r.importsOut)

	return nil
}
