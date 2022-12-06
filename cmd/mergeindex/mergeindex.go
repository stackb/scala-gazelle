package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
	"github.com/stackb/scala-gazelle/pkg/jarindex"
)

const debug = false

var (
	outputFile       string
	predefinedLabels string
	preferredLabels  string
)

func main() {
	log.SetPrefix("mergeindex: ")
	log.SetFlags(0) // don't print timestamps

	args := os.Args[1:]
	if len(args) == 1 && strings.HasPrefix(args[0], "@") {
		paramsFile := args[0][1:]
		var err error
		args, err = readParamsFile(paramsFile)
		if err != nil {
			log.Fatalln("failed to read params file:", paramsFile, err)
		}
	}

	files, err := parseFlags(args)
	if err != nil {
		log.Fatal(err)
	}

	index, err := merge(files...)
	if err != nil {
		log.Fatal(err)
	}

	if err := jarindex.WriteJarIndexFile(outputFile, index); err != nil {
		log.Fatal(err)
	}
}

func readParamsFile(filename string) ([]string, error) {
	params := []string{}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		params = append(params, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return params, nil
}

func parseFlags(args []string) (files []string, err error) {
	fs := flag.NewFlagSet("mergeindex", flag.ExitOnError) // flag.ContinueOnError
	fs.StringVar(&predefinedLabels, "predefined", "", "a comma-separated list of labels to be considered predefined")
	fs.StringVar(&preferredLabels, "preferred", "", "a comma-separated list of labels to be considered preferred")
	fs.StringVar(&outputFile, "output_file", "", "the output file to write")
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: mergeindex @PARAMS_FILE | mergeindex OPTIONS FILES")
		fs.PrintDefaults()
	}
	if err = fs.Parse(args); err != nil {
		return
	}

	if outputFile == "" {
		log.Fatal("-output_file is required")
	}

	files = fs.Args()
	if len(files) == 0 {
		err = fmt.Errorf("mergeindex positional args should be a non-empty list of jarindex.{pb|json} files to merge")
	}

	return
}

func merge(filenames ...string) (*jipb.JarIndex, error) {
	jars := make([]*jipb.JarFile, 0, len(filenames))
	for _, filename := range filenames {
		idx, err := jarindex.ReadJarIndexFile(filename)
		if err != nil {
			return nil, err
		}
		jars = append(jars, idx.JarFile...)
		if debug {
			if err := writeJarIndexJarFileJSONFiles(idx); err != nil {
				return nil, err
			}
		}
	}

	var predefined []string
	if predefinedLabels != "" {
		predefined = strings.Split(predefinedLabels, ",")
	}

	log.Println("predefined:", predefined)

	index, err := jarindex.MergeJarFiles(func(format string, args ...interface{}) {
		log.Printf("warning: "+format, args...)
	}, predefined, jars)
	if err != nil {
		return nil, err
	}

	index.Predefined = predefined

	return index, nil
}

func writeJarIndexJarFileJSONFiles(idx *jipb.JarIndex) error {
	for _, file := range idx.JarFile {
		if err := writeJarFileJSONFile(file); err != nil {
			return err
		}
	}
	return nil
}

func writeJarFileJSONFile(file *jipb.JarFile) error {
	jarFilename := "/tmp/" + filepath.Base(file.Filename) + ".json"
	if err := jarindex.WriteJarFileFile(jarFilename, file); err != nil {
		return err
	}
	log.Println("Wrote:", jarFilename)
	return nil
}
