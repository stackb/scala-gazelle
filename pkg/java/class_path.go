package java

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

const (
	CLASS_FILE_SUFFIX = ".class"
	JAR_FILE_SUFFIX   = ".jar"
)

type ClassPathEntry interface {
	readClass(className string) ([]byte, error)
	String() string
}

type DirectoryClassPathEntry struct {
	directory string
}

func (this *DirectoryClassPathEntry) String() string {
	return this.directory
}

func (this *DirectoryClassPathEntry) readClass(className string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(this.directory, className+CLASS_FILE_SUFFIX))
}

type JarClassPathEntry struct {
	jarFile string
}

func NewJarClassPathEntry(jarFile string) *JarClassPathEntry {
	return &JarClassPathEntry{jarFile}
}

func (this *JarClassPathEntry) String() string {
	return this.jarFile
}

// Visit takes a function that accepts the zip file entry
func (this *JarClassPathEntry) Visit(accept func(f *zip.File, clazz *ClassFile) error) error {
	r, err := zip.OpenReader(this.jarFile)
	if err != nil {
		return err
	}

	defer r.Close()
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".class") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}

		defer rc.Close()
		bytecode, err := ioutil.ReadAll(rc)
		if err != nil {
			return err
		}

		// log.Printf("reading class %v: %d", f.Name, len(bytecode))

		classfile := &ClassFile{}
		classfile.read(bytecode)

		if err := accept(f, classfile); err != nil {
			return err
		}
	}
	return nil
}

func (this *JarClassPathEntry) readClass(className string) ([]byte, error) {
	r, err := zip.OpenReader(this.jarFile)
	if err != nil {
		return nil, err
	}

	defer r.Close()
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		defer rc.Close()
		bytecode, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}

		return bytecode, nil
	}
	return nil, errors.New(fmt.Sprintf("Class not found %s in %s", className, this.jarFile))
}

type ClassPath struct {
	classPathEntries []ClassPathEntry
}

func NewClassPath(classPathStr string) *ClassPath {
	segs := strings.Split(classPathStr, ":")
	// filter empty string
	var entries []string
	for _, str := range segs {
		if str != "" {
			entries = append(entries, str)
		}
	}
	classPath := &ClassPath{make([]ClassPathEntry, len(entries))}
	for i, entry := range entries {
		path, err := filepath.Abs(entry)
		if err != nil {
			log.Fatal("Not a legal path %s", entry)
		}
		if strings.HasSuffix(entry, JAR_FILE_SUFFIX) {
			classPath.classPathEntries[i] = &JarClassPathEntry{path}
		} else {
			classPath.classPathEntries[i] = &DirectoryClassPathEntry{path}
		}
	}
	return classPath
}

func (this *ClassPath) String() string {
	entries := make([]string, len(this.classPathEntries))
	for i, entry := range this.classPathEntries {
		entries[i] = entry.String()
	}
	return strings.Join(entries, ":")
}

func (this *ClassPath) ReadClass(className string) ([]byte, error) {
	for _, entry := range this.classPathEntries {
		bytecode, err := entry.readClass(className)
		if err == nil {
			return bytecode, err
		}
	}

	return nil, errors.New(fmt.Sprintf("Class cannot be read from classpath: %s", this.String()))
}
