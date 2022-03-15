package main

import (
	"archive/zip"
	"flag"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/stackb/scala-gazelle/pkg/index"
	"github.com/stackb/scala-gazelle/pkg/java"
)

const (
	debug = false
)

var (
	isAnonymous = regexp.MustCompile(`^.*\$[0-9]$`)

	inputFile  string
	outputFile string
)

func main() {
	log.SetPrefix("jarindexer: ")
	log.SetFlags(0) // don't print timestamps

	fs := flag.NewFlagSet("jarindexer", flag.ContinueOnError)
	fs.StringVar(&inputFile, "input_file", "", "the input configuration file")
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if inputFile == "" {
		log.Fatal("-input_file is required")
	}
	if outputFile == "" {
		log.Fatal("-output_file is required")
	}
	if debug {
		index.ListFiles(".")
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	spec, err := index.ReadJarSpec(inputFile)
	if err != nil {
		return err
	}
	if err := parseJarFile(spec.Filename, spec); err != nil {
		log.Printf("warning: could not parse %s: %v", spec.Filename, err)
	}

	sort.Strings(spec.Classes)

	if err := index.WriteJSONFile(outputFile, spec); err != nil {
		return err
	}
	return nil
}

func parseJarFile(filename string, spec *index.JarSpec) error {
	if debug {
		log.Println("Parsing jar file:", filename)
	}
	pkgs := make(map[string]bool)

	symbols := make([]string, 0)
	symbolIndex := make(map[string]int)

	entry := java.NewJarClassPathEntry(filename)
	if err := entry.Visit(func(f *zip.File, c *java.ClassFile) error {
		if c.IsSynthetic() {
			if debug {
				log.Println("skipping synthetic class:", f.Name, c.Name())
			}
			return nil
		}
		name := c.Name()
		if debug {
			log.Println("Visiting class:", f.Name, name)
		}

		// exclude Main$ scala classes
		if strings.HasSuffix(name, "$") {
			if debug {
				log.Println("skipping scala singleton class:", f.Name, c.Name())
			}
			return nil
		}

		// exclude anonymous classes like 'com/google/protobuf/Int32Value$1'
		if isAnonymous.MatchString(name) {
			if debug {
				log.Println("skipping anonymous class:", f.Name, c.Name())
			}
			return nil
		}

		// transform "org/json4s/package$MappingException.class" ->
		// "org/json4s/MappingException.class" so that
		// "org.json4s.MappingException" is resolveable.
		base := path.Base(name)
		if strings.HasPrefix(base, "package$") {
			name = path.Join(path.Dir(name), base[len("package$"):])
		}

		name = convertClassName(name)

		// use the scala convention to generate a class for the package to
		// populate the packages index.  This might not be correct.
		if strings.HasSuffix(name, ".package") {
			pkgs[strings.TrimSuffix(name, ".package")] = true
		} else {
			spec.Classes = append(spec.Classes, name)
			if pkgName, ok := classPackageName(name); ok {
				pkgs[pkgName] = true
			}
		}

		for _, pkgName := range c.PackageNames() {
			pkgs[convertPackageName(pkgName)] = true
		}

		superClass := c.SuperClassName()
		// save space by leaving out Object
		if superClass != "" && superClass != "java/lang/Object" {
			if spec.Extends == nil {
				spec.Extends = make(map[string]string)
			}
			spec.Extends[name] = convertClassName(superClass)
		}

		classes := c.Symbols()
		classFile := &index.ClassFileSpec{Name: name}
		if len(classes) > 0 {
			classFile.Classes = make([]int, len(classes))
			for i, name := range classes {
				className := convertClassName(name)
				if index, ok := symbolIndex[className]; ok {
					classFile.Classes[i] = index
				} else {
					index := len(symbols)
					symbols = append(symbols, className)
					symbolIndex[className] = index
					classFile.Classes[i] = index
				}
			}
		}
		spec.Files = append(spec.Files, classFile)

		return nil
	}); err != nil {
		return err
	}

	packages := make([]string, 0, len(pkgs))
	for p := range pkgs {
		packages = append(packages, p)
	}
	sort.Strings(packages)
	spec.Packages = packages
	spec.Symbols = symbols
	return nil
}

func convertClassName(name string) string {
	name = strings.Replace(name, "/", ".", -1)
	// name = strings.Replace(name, "$", ".", -1) // TODO(pcj): is this correct to do this with the inner classes?
	return name
}

func convertPackageName(name string) string {
	return strings.Replace(name, "/", ".", -1)
}

func classPackageName(name string) (string, bool) {
	lastDot := strings.LastIndex(name, ".")
	if lastDot <= 0 {
		return "", false
	}

	pkg := name[0:lastDot]
	return pkg, true
}
