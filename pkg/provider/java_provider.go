package provider

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const debugUnresolvedSuperclass = false

// JavaProvider is a provider of symbols for a set of jarindex protos.
type JavaProvider struct {
	jarindexFiles collections.StringSlice

	scope   resolver.Scope
	byLabel map[label.Label]*jipb.JarFile
	// classSymbols is map a *.ClassFile to it's symbol
	classSymbols map[*jipb.ClassFile]*resolver.Symbol
	// preferred packages
	preferred map[string]label.Label
}

// NewJavaProvider constructs a new provider.
func NewJavaProvider() *JavaProvider {
	return &JavaProvider{
		byLabel:       make(map[label.Label]*jipb.JarFile),
		jarindexFiles: make(collections.StringSlice, 0),
		classSymbols:  make(map[*jipb.ClassFile]*resolver.Symbol),
		preferred:     make(map[string]label.Label),
	}
}

// Name implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) Name() string {
	return "java"
}

// RegisterFlags implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.Var(&p.jarindexFiles, "javaindex_file", "path to javaindex.pb or javaindex.json file")
}

// CheckFlags implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, scope resolver.Scope) error {
	p.scope = scope

	for _, filename := range p.jarindexFiles {
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(c.WorkDir, filename)
		}
		if err := p.readJarIndex(c, filename); err != nil {
			return err
		}
	}

	return nil
}

// OnResolve implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) OnResolve() error {
	classFiles := sortedSymbolClassfiles(p.classSymbols)
	for _, classFile := range classFiles {
		symbol := p.classSymbols[classFile]
		for _, superclass := range append(classFile.Superclasses, classFile.Interfaces...) {
			if resolved, ok := p.scope.GetSymbol(superclass); ok {
				symbol.Require(resolved)
			} else if debugUnresolvedSuperclass {
				log.Printf("Unresolved superclass %s of %s", superclass, classFile.Name)
			}
		}
	}
	return nil
}

// OnEnd implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) OnEnd() error {
	return nil
}

// CanProvide implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) CanProvide(dep *resolver.ImportLabel, expr build.Expr, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	if _, ok := p.byLabel[dep.Label]; ok {
		return true
	}
	return false
}

// GetPreferredDeps exposes the preferred package mapping for a conflict resolver.
func (p *JavaProvider) GetPreferredDeps() map[string]label.Label {
	return p.preferred
}

func (p *JavaProvider) readJarIndex(c *config.Config, filename string) error {
	var index jipb.JarIndex
	if err := protobuf.ReadFile(filename, &index); err != nil {
		return fmt.Errorf("reading jar index file %s: %v (from %s)", filename, err, c.RepoRoot)
	}

	isPredefined := make(map[label.Label]bool)
	for _, v := range index.Predefined {
		lbl, err := label.Parse(v)
		if err != nil {
			return fmt.Errorf("bad predefined label %q: %v", v, err)
		}
		isPredefined[lbl] = true
	}

	for k, v := range index.Preferred {
		dep, err := label.Parse(v)
		if err != nil {
			return fmt.Errorf("malformed preferred label %s: %v", v, err)
		}
		p.preferred[k] = dep
	}

	for _, jarFile := range index.JarFile {
		if err := p.readJarFile(jarFile, isPredefined); err != nil {
			return err
		}
	}

	return nil
}

func (p *JavaProvider) readJarFile(jarFile *jipb.JarFile, isPredefined map[label.Label]bool) error {
	if jarFile.Filename == "" {
		log.Panicf("jarFile must have a name: %+v", jarFile)
	}

	var from label.Label
	if jarFile.Label != "" {
		var err error
		from, err = label.Parse(jarFile.Label)
		if err != nil {
			return fmt.Errorf("%s: parsing label %q: %v", jarFile.Filename, jarFile.Label, err)
		}
	}
	p.byLabel[from] = jarFile

	if isPredefined[from] {
		from = label.NoLabel
	}

	for _, pkg := range jarFile.PackageName {
		p.putSymbol(sppb.ImportType_PACKAGE, pkg, from)
	}

	for _, classFile := range jarFile.ClassFile {
		p.readClassFile(classFile, from)
	}

	return nil
}

func (p *JavaProvider) readClassFile(classFile *jipb.ClassFile, from label.Label) {
	impType := sppb.ImportType_CLASS
	if classFile.IsInterface {
		impType = sppb.ImportType_INTERFACE
	}
	symbol := p.putSymbol(impType, classFile.Name, from)
	if len(classFile.Superclasses) > 0 || len(classFile.Interfaces) > 0 {
		p.classSymbols[classFile] = symbol
	}
}

func (p *JavaProvider) putSymbol(impType sppb.ImportType, imp string, from label.Label) *resolver.Symbol {
	symbol := resolver.NewSymbol(impType, imp, p.Name(), from)
	p.scope.PutSymbol(symbol)
	return symbol
}

func sortedSymbolClassfiles(classSymbols map[*jipb.ClassFile]*resolver.Symbol) []*jipb.ClassFile {
	classFiles := make([]*jipb.ClassFile, 0, len(classSymbols))
	for classFile := range classSymbols {
		classFiles = append(classFiles, classFile)
	}
	sort.Slice(classFiles, func(i, j int) bool {
		a := classFiles[i]
		b := classFiles[j]
		return a.Name < b.Name
	})
	return classFiles
}
