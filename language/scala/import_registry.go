package scala

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/RoaringBitmap/roaring"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/stackb/scala-gazelle/pkg/collections"
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
	Disambiguate(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string, from label.Label, labels []label.Label) ([]label.Label, error)
	// Get the label that provides the given import
	Provider(imp string) (label.Label, bool)
	// Completions returns concrete symbols under the given prefix
	Completions(prefix string) map[string]label.Label
	// ResolveName implements NameResolver
	ResolveName(name string) (string, bool)
	// ResolveLabel implements LabelResolver
	ResolveLabel(name string) (label.Label, bool)
	// TransitiveImports calculates the dependencies of the given deps.
	TransitiveImports(deps []string) (resolved, unresolved []string)
}

// NameResolver is a function that takes a symbol name.  So for 'LazyLogging' it
// should return 'com.typesafe.scalalogging.LazyLogging'.
type NameResolver func(name string) (string, bool)

// LabelResolver is a function that takes a fully-qualified import.  So for
// 'com.typesafe.scalalogging.LazyLogging' it should return
// '@maven//:com_typesafe_scala_logging_scala_logging_2_12'.
type LabelResolver func(name string) (label.Label, bool)

// DependencyRecorder is a function that records a dependency between src and
// dst.  For example, class java.util.ArrayList (src) has a dependency on
// java.util.List (dst).
type DependencyRecorder func(src, dst string)

type dependencyMatrix map[uint32]*roaring.Bitmap

func newImportRegistry(sourceRuleRegistry ScalaSourceRuleRegistry, classFileRegistry resolve.CrossResolver, scalaCompiler ScalaCompiler) *importRegistry {
	return &importRegistry{
		importsOut:         "/tmp/scala-gazelle-imports.csv",
		sourceRuleRegistry: sourceRuleRegistry,
		classFileRegistry:  classFileRegistry,
		scalaCompiler:      scalaCompiler,
		provides:           make(map[label.Label][]string),
		imports:            make(map[string]label.Label),
		classes:            make(map[string][]label.Label),
		symbols:            NewSymbolTable(),
		dependencies:       make(dependencyMatrix),
	}
}

// importRegistry implements ScalaImportRegistry.
type importRegistry struct {
	// sourceRuleRegistry is used to assist with disambigation
	sourceRuleRegistry ScalaSourceRuleRegistry
	// classFileRegistry is used to assist with disambigation
	classFileRegistry resolve.CrossResolver
	// scalaCompiler is used to assist with disambigation
	scalaCompiler ScalaCompiler
	// provides is a mapping of 'from' labels representing the concrete types that from provides.
	provides map[label.Label][]string
	// imports is a mapping of an import to the label that provides it, the
	// inverse map of provides.
	imports map[string]label.Label
	// classes is a mapping from the last symbol in a import to the labels that provides them.
	classes map[string][]label.Label
	// importsOut is the name of a file to write the index to
	importsOut string
	// symbols is the symbol table of known classes
	symbols *SymbolTable
	// dependencies is a sparse matrix representing class dependencies in the symbol table.
	dependencies dependencyMatrix
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

func (ir *importRegistry) ResolveName(name string) (string, bool) {
	suffix := "." + name
	for imp := range ir.imports {
		if strings.HasSuffix(imp, suffix) {
			return imp, true
		}
	}
	return "", false
}

func (ir *importRegistry) ResolveLabel(imp string) (label.Label, bool) {
	from, ok := ir.imports[imp]
	return from, ok
}

func (ir *importRegistry) Provider(imp string) (label.Label, bool) {
	from, ok := ir.imports[imp]
	return from, ok
}

func (ir *importRegistry) Provides(l label.Label, imports []string) {
	ir.provides[l] = append(ir.provides[l], imports...)
}

// Depends records a compile-time dependency of src on dst.
func (ir *importRegistry) Depends(src, dst string) {
	i := ir.symbols.Add(src)
	j := ir.symbols.Add(dst)
	deps, ok := ir.dependencies[i]
	if !ok {
		deps = roaring.New()
		ir.dependencies[i] = deps
	}
	deps.Add(j)
	log.Printf("depends: %s -> %s", src, dst)
}

func (ir *importRegistry) TransitiveImports(deps []string) (resolved, unresolved []string) {
	transitive := roaring.New()

	for _, dep := range deps {
		// log.Println("resolving transitive imports of", dep)
		if id, ok := ir.symbols.Get(dep); ok {
			ir.importsFor(id, transitive, true)
		} else {
			unresolved = append(unresolved, dep)
		}
	}

	resolved = ir.symbols.ResolveAll(&roaringBitSet{transitive})

	return
}

func (ir *importRegistry) importsFor(dep uint32, transitive *roaring.Bitmap, allTransitive bool) {
	seen := make(map[uint32]struct{})
	stack := make(collections.UInt32Stack, 0)
	stack.Push(dep)

	for !stack.IsEmpty() {
		current, _ := stack.Pop()

		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		// transitive.Add(current)

		deps, ok := ir.dependencies[current]
		if !ok {
			continue
		}
		transitive.Or(deps)

		if allTransitive {
			it := deps.Iterator()
			for it.HasNext() {
				stack.Push(it.Next())
			}
		}
	}
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

func (ir *importRegistry) Disambiguate(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string, from label.Label, labels []label.Label) ([]label.Label, error) {
	debug := false

	fail := func(reason string) ([]label.Label, error) {
		return labels, fmt.Errorf(`%[1]q is ambiguous (symbol is provided by multiple rules %[2]v)

Failed to automatically disambiguate it to a single label, or matching to an unambiguous list. 

Reason: %[4]s

Rule where the error originated: %[3]v

Possible solutions:

1. Use '# gazelle:resolve scala scala %[1]s LABEL' to pick one.
   - Alternatively: '# gazelle:override scala glob GLOB LABEL' .
2. If this is a wildcard import, remove the wildcard and be explicit.
3. If this is a jar, add the desired choice to the 'preferred' attribute (java_index rule).
4. If this is a jar, remove duplicate providers from the 'deps' attribute (java_index rule).
`, imp.Imp, labels, from, reason)
	}

	// step 1: check override specs
	if sc := getScalaConfig(c); sc != nil {
		for _, override := range sc.overrides {
			// log.Printf("%v: check match override %q with override: %v", from, imp, override.imp.Imp)
			if ok, _ := doublestar.Match(override.imp.Imp, imp.Imp); ok {
				if debug {
					log.Printf("%v: disambiguated %q with override: %v", from, imp.Imp, override.dep)
				}
				return []label.Label{override.dep}, nil
			}
		}
	}

	if debug {
		log.Printf("%v: disambiguating %q, candidate labels: %v", from, imp.Imp, labels)
	}

	// step 1a: check if this import is provided by a jar file.  If so, no point
	// in trying to resolve it via srcs.
	if result := ir.classFileRegistry.CrossResolve(c, ix, imp, lang); len(result) > 0 {
		return fail("multiple jar files provide the import")
	}

	// step 2: get completion symbols for the import.  For example, if we had
	// the import 'java.util._', get the set (java.util.List, java.util.Map,
	// ...)
	completions := ir.Completions(imp.Imp)
	if len(completions) == 0 {
		return fail(imp.Imp + " did not expand to any known concrete types")
	}

	if debug {
		for i, c := range completions {
			log.Printf("completion universe member %q: %v", i, c)
		}
	}

	// step 3: gather the list of srcs in 'from' and filter them such that
	// only those that explicitly use the import are retained.
	rule, ok := ir.sourceRuleRegistry.GetScalaRule(from)
	if !ok {
		return fail(fmt.Sprintf("rule registry: unknown rule %v", from))
	}

	files := []*index.ScalaFileSpec{}
	for _, file := range rule.Srcs {
		for _, i := range file.Imports {
			if imp.Imp == i {
				files = append(files, file)
				break
			}
		}
	}
	if len(files) == 0 {
		return fail(fmt.Sprintf("did not find a source file in %v that actually imports %q; can't use source file symbols to help disambiguate futher", from, imp.Imp))
	}

	// step 4: use the scala compiler to list the unknown types in the source file.
	types := make(map[string]bool)

	for _, file := range files {
		compilation, err := ir.scalaCompiler.Compile(c.RepoRoot, file.Filename)
		if err != nil {
			return fail(fmt.Sprintf("scala compiler error %q: %v", file.Filename, err))
		}
		for _, sym := range compilation.NotFound {
			types[sym.Name] = true
		}
		for _, sym := range compilation.NotMember {
			if sym.Package == imp.Imp {
				types[sym.Name] = true
			}
		}
		// augment the types map with any imports they specifically named in an import.
		for _, fileImp := range file.Imports {
			if strings.HasPrefix(fileImp, imp.Imp) {
				sym := strings.TrimPrefix(fileImp[len(imp.Imp):], ".")
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

	// step 5: process all the unknown types in srcs.  If we find a completion
	// match, we assume they are using this actual symbol.
	match := make(map[label.Label]bool)

	for sym := range types {
		from, ok := completions[sym]
		if ok {
			match[from] = true
		}
	}

	if len(match) == 0 {
		return fail(fmt.Sprintf("unable to find a completion match in %v", types))
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
