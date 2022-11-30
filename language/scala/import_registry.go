package scala

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/RoaringBitmap/roaring"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/stackb/scala-gazelle/pkg/collections"
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
	TransitiveImports(deps []string, depth int) (resolved, unresolved []string)
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
type DependencyRecorder func(src, dst, kind string)

type dependencyMatrix map[uint32]*roaring.Bitmap

func newImportRegistry(classFileRegistry ScalaJarResolver, scalaCompiler ScalaCompiler) *importRegistry {
	return &importRegistry{
		importsOut:        "/tmp/scala-gazelle-imports.csv",
		classFileRegistry: classFileRegistry,
		scalaCompiler:     scalaCompiler,
		provides:          make(map[label.Label][]string),
		imports:           make(map[string]label.Label),
		classes:           make(map[string][]label.Label),
		symbols:           NewSymbolTable(),
		dependencies:      make(dependencyMatrix),
		depEdges:          make(map[string]string),
	}
}

// importRegistry implements ScalaImportRegistry.
type importRegistry struct {
	// classFileRegistry is used to assist with disambigation
	classFileRegistry ScalaJarResolver
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
	// depEdges records the kind of edge between i and j
	depEdges map[string]string
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

// AddDependency records a compile-time dependency of src on dst.  The kind
// argument can be any string prefix, typically, 'import', 'file', etc.
func (ir *importRegistry) EdgeKind(src, dst uint32) string {
	return ir.depEdges[fmt.Sprintf("%d.%d", src, dst)]
}

// AddDependency records a compile-time dependency of src on dst.  The kind
// argument can be any string prefix, typically, 'import', 'file', etc.
func (ir *importRegistry) AddDependency(src, dst, kind string) {
	if src == "" {
		return
	}
	i := ir.symbols.Add(src)
	// if the caller has used an empty dst, this is a means to just add the
	// symbol to the symbol-table.
	if dst == "" {
		return
	}

	deps, ok := ir.dependencies[i]
	if !ok {
		deps = roaring.New()
		ir.dependencies[i] = deps
	}

	j := ir.symbols.Add(dst)
	deps.Add(j)

	edgeKey := fmt.Sprintf("%d.%d", i, j)
	ir.depEdges[edgeKey] = kind

	// log.Printf("importRegistry.depends: %s --[%s]--> %s", src, kind, dst)
}

func (ir *importRegistry) Previous(dst string) *roaring.Bitmap {
	prev := roaring.New()

	id, ok := ir.symbols.Get(dst)
	if !ok {
		return prev
	}

	suffix := fmt.Sprintf(".%d", id)

	for k := range ir.depEdges {
		if strings.HasSuffix(k, suffix) {
			value, err := strconv.Atoi(k[:len(k)-len(suffix)])
			if err != nil {
				log.Panicf("malformed edge key: %q: %v", k, err)
			}
			prev.Add(uint32(value))
		}
	}

	return prev
}

func (ir *importRegistry) DirectImports(dep string) (resolved, unresolved []string) {
	transitive := roaring.New()

	// log.Println("resolving transitive imports of", dep)
	if id, ok := ir.symbols.Get(dep); ok {
		ir.depsFor(id, transitive, 1)
	} else {
		unresolved = append(unresolved, dep)
	}

	resolved = ir.symbols.ResolveAll(&roaringBitSet{transitive}, "imp")

	return
}

func (ir *importRegistry) TransitiveImports(deps []string, depth int) (resolved, unresolved []string) {
	transitive := roaring.New()
	seen := make(map[uint32]struct{})

	for _, dep := range deps {
		if id, ok := ir.symbols.Get(dep); ok {
			ir.depsOf(id, transitive, seen, depth)
		} else {
			unresolved = append(unresolved, dep)
		}
	}

	resolved = ir.symbols.ResolveAll(&roaringBitSet{transitive}, "imp")

	return
}

func (ir *importRegistry) depsFor(dep uint32, transitive *roaring.Bitmap, depth int) {
	seen := make(map[uint32]struct{})

	stack := make(collections.UInt32Stack, 0)
	stack.Push(dep)

	for !stack.IsEmpty() {
		current, _ := stack.Pop()
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		deps, ok := ir.dependencies[current]
		if !ok {
			continue
		}
		transitive.Or(deps)

		if depth < 0 || len(stack) <= depth {
			it := deps.Iterator()
			for it.HasNext() {
				stack.Push(it.Next())
			}
		}
	}
}

func (ir *importRegistry) depsOf(dep uint32, transitive *roaring.Bitmap, seen map[uint32]struct{}, depth int) {
	if _, ok := seen[dep]; ok {
		return
	}
	seen[dep] = struct{}{}

	deps, ok := ir.dependencies[dep]
	if !ok {
		return
	}
	transitive.Or(deps)

	depth--
	if depth == 0 {
		return
	}

	it := deps.Iterator()
	for it.HasNext() {
		ir.depsOf(it.Next(), transitive, seen, depth)
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
	return nil, fmt.Errorf("Disambiguate is no longer implemented.")
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

	return nil
}

func findPackageSymbolCompletion(registry ScalaImportRegistry, packages []string, want string) (string, label.Label) {
	for _, pkg := range packages {
		completions := registry.Completions(pkg)
		for got, from := range completions {
			if got == want {
				return got, from
			}
		}
	}
	return "", label.NoLabel
}
