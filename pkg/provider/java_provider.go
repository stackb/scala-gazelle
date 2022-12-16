package provider

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

const debugUnresolvedSuperclass = false
const javaName = "java"

// JavaProvider is a provider of symbols for a set of jarindex protos.
type JavaProvider struct {
	jarindexFiles collections.StringSlice

	scope   resolver.Scope
	byLabel map[label.Label]*jipb.JarFile
	// classSymbols is map a *.ClassFile to it's symbol
	classSymbols map[*jipb.ClassFile]*resolver.Symbol
}

// NewJavaProvider constructs a new provider.
func NewJavaProvider() *JavaProvider {
	return &JavaProvider{
		byLabel:       make(map[label.Label]*jipb.JarFile),
		jarindexFiles: make(collections.StringSlice, 0),
		classSymbols:  make(map[*jipb.ClassFile]*resolver.Symbol),
	}
}

// Name implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) Name() string {
	return javaName
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
		if err := p.readJarIndex(filename); err != nil {
			return err
		}
	}

	return nil
}

// OnResolve implements part of the resolver.SymbolProvider interface.
func (p *JavaProvider) OnResolve() error {
	for classFile, symbol := range p.classSymbols {
		for _, superclass := range append(classFile.Superclasses, classFile.Interfaces...) {
			if resolved, ok := p.scope.GetSymbol(superclass); ok {
				symbol.Requires = append(symbol.Requires, resolved)
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
func (p *JavaProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	if _, ok := p.byLabel[dep]; ok {
		return true
	}
	return false
}

func (p *JavaProvider) readJarIndex(filename string) error {
	var index jipb.JarIndex
	if err := protobuf.ReadFile(filename, &index); err != nil {
		return fmt.Errorf("reading %s: %v", filename, err)
	}

	isPredefined := make(map[label.Label]bool)
	for _, v := range index.Predefined {
		lbl, err := label.Parse(v)
		if err != nil {
			return fmt.Errorf("bad predefined label %q: %v", v, err)
		}
		isPredefined[lbl] = true
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
	symbol := resolver.NewSymbol(impType, imp, javaName, from)
	p.scope.PutSymbol(symbol)
	return symbol
}
