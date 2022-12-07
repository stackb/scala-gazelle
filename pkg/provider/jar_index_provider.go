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
	"github.com/stackb/scala-gazelle/pkg/jarindex"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

// JarIndexProvider is a provider of known imports for a set of
// jar index protos.
type JarIndexProvider struct {
	jarindexFiles collections.StringSlice

	knownImportRegistry resolver.KnownImportRegistry
	byLabel             map[label.Label]*jipb.JarFile
}

// NewJarIndexProvider constructs a new provider.  The lang/impLang
// arguments are used to fetch the provided imports in the given importProvider
// struct.
func NewJarIndexProvider() *JarIndexProvider {
	return &JarIndexProvider{
		byLabel:       make(map[label.Label]*jipb.JarFile),
		jarindexFiles: make(collections.StringSlice, 0),
	}
}

// Name implements part of the resolver.KnownImportProvider interface.
func (p *JarIndexProvider) Name() string {
	return "jarindex"
}

// RegisterFlags implements part of the resolver.KnownImportProvider interface.
func (p *JarIndexProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.Var(&p.jarindexFiles, "jarindex_file", "path to jarindex.pb or jarindex.json file")
}

// CheckFlags implements part of the resolver.KnownImportProvider interface.
func (p *JarIndexProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, registry resolver.KnownImportRegistry) error {
	p.knownImportRegistry = registry

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

// OnResolve implements part of the resolver.KnownImportProvider interface.
func (p *JarIndexProvider) OnResolve() {

}

// CanProvide implements part of the resolver.KnownImportProvider interface.
func (p *JarIndexProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	if _, ok := p.byLabel[dep]; ok {
		return true
	}
	return false
}

func (p *JarIndexProvider) readJarIndex(filename string) error {
	index, err := jarindex.ReadJarIndexFile(filename)
	if err != nil {
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

	// for _, v := range index.Preferred {
	// 	lbl, err := label.Parse(v)
	// 	if err != nil {
	// 		return fmt.Errorf("bad preferred label %q: %v", v, err)
	// 	}
	// 	p.preferred[lbl] = true
	// }

	for _, jarFile := range index.JarFile {
		if err := p.readJarFile(jarFile, isPredefined); err != nil {
			return err
		}
	}

	return nil
}

func (p *JarIndexProvider) readJarFile(jarFile *jipb.JarFile, isPredefined map[label.Label]bool) error {
	if jarFile.Filename == "" {
		log.Panicf("unnamed jar? %+v", jarFile)
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
		p.putKnownImport(sppb.ImportType_PACKAGE, pkg, from)
	}

	for _, classFile := range jarFile.ClassFile {
		p.putKnownImport(sppb.ImportType_CLASS, classFile.Name, from)
	}

	for _, classFile := range jarFile.ClassFile {
		p.readClassFile(classFile, from)
	}

	return nil
}

func (p *JarIndexProvider) readClassFile(classFile *jipb.ClassFile, from label.Label) {
	impType := sppb.ImportType_CLASS
	if classFile.IsInterface {
		impType = sppb.ImportType_INTERFACE
	}
	p.putKnownImport(impType, classFile.Name, from)
}

func (p *JarIndexProvider) putKnownImport(impType sppb.ImportType, imp string, from label.Label) {
	p.knownImportRegistry.PutKnownImport(resolver.NewKnownImport(impType, imp, p.Name(), from))
}
