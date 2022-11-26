package mergeindex

import (
	"fmt"
	"io/ioutil"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
)

type warnFunc func(format string, args ...interface{})

func MergeJarFiles(warn warnFunc, predefined []string, jarFiles []*jarindex.JarFile) (*jarindex.JarIndex, error) {
	var index jarindex.JarIndex

	// jarLabels is used to prevent duplicate entries for a given jar.
	labels := make(map[string]bool)

	// providersByClass is used to check if more than one label provides a given
	// class.
	providersByClass := make(map[string]*jarindex.ClassFileProvider)

	// predefinedLabels do not need to be resolved
	predefinedLabels := make(map[string]struct{})
	for _, l := range predefined {
		predefinedLabels[l] = struct{}{}
	}

	// predefinedSymbols is the set of symbols we can remove from each class
	// files' list of symbols; these will never need to be resolved.
	predefinedSymbols := map[string]struct{}{
		"java.lang.Object": {},
	}

	for _, jar := range jarFiles {
		if labels[jar.Label] {
			warn("duplicate jar label: %s", jar.Label)
			continue
		}
		labels[jar.Label] = true
		visitJarFile(warn, jar, predefinedLabels, predefinedSymbols, providersByClass)
		index.JarFile = append(index.JarFile, jar)
	}

	// remove predefined symbols
	for _, jar := range index.JarFile {
		for _, file := range jar.ClassFile {
			var resolvable []string
			for _, sym := range file.Symbols {
				if _, ok := predefinedSymbols[sym]; ok {
					continue
				}
				resolvable = append(resolvable, sym)
			}
			file.Symbols = resolvable
		}
	}

	for classname, providers := range providersByClass {
		if len(providers.Label) > 1 {
			warn("class is provided by more than one label: %s: %v", classname, providers.Label)
		}
	}

	return &index, nil
}

func visitJarFile(
	warn warnFunc,
	jar *jarindex.JarFile,
	predefinedLabels, predefinedSymbols map[string]struct{},
	providersByClass map[string]*jarindex.ClassFileProvider,
) {

	if jar.Label == "" {
		warn("missing jar label: %s", jar.Filename)
		return
	}
	if jar.Filename == "" {
		warn("missing jar filename: %s", jar.Label)
		return
	}

	// log.Println("---", jar.Label, "---")

	if _, ok := predefinedLabels[jar.Label]; ok {
		// TODO: consider only recording packages, not classes
		for _, file := range jar.ClassFile {
			predefinedSymbols[file.Name] = struct{}{}
		}
	}

	for _, classFile := range jar.ClassFile {
		providers, found := providersByClass[classFile.Name]
		if !found {
			providers = &jarindex.ClassFileProvider{Class: classFile.Name}
			providersByClass[classFile.Name] = providers
			// log.Println(classFile.Name, "is provided by:", jar.Label)
		}
		providers.Label = append(providers.Label, jar.Label)
	}

}

func ReadJarIndexProtoFile(filename string) (*jarindex.JarIndex, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read jarindex file %q: %w", filename, err)
	}
	index := jarindex.JarIndex{}
	if err := proto.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal jarindex proto: %w", err)
	}
	return &index, nil
}

func WriteJarIndexProtoFile(filename string, index *jarindex.JarIndex) error {
	data, err := proto.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarindex proto: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarindex proto: %w", err)
	}
	return nil
}

func WriteJarIndexJSONFile(filename string, index *jarindex.JarIndex) error {
	data, err := protojson.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarindex json: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarindex json: %w", err)
	}
	return nil
}

func ReadJarFileProtoFile(filename string) (*jarindex.JarFile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read jarfile file %q: %w", filename, err)
	}
	index := jarindex.JarFile{}
	if err := proto.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal jarfile proto: %w", err)
	}
	return &index, nil
}

func WriteJarFileProtoFile(filename string, index *jarindex.JarFile) error {
	data, err := proto.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarfile proto: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarfile proto: %w", err)
	}
	return nil
}

func WriteJarFileJSONFile(filename string, index *jarindex.JarFile) error {
	data, err := protojson.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarfile json: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarfile json: %w", err)
	}
	return nil
}
