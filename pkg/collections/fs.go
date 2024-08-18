package collections

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ListFiles is a convenience debugging function to log the files under a given dir.
func ListFiles(dir string) error {
	log.Println("Listing files under " + dir)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		log.Println(path)
		return nil
	})
}

// CollectFiles is a convenience function to gather files under a directory.
func CollectFiles(dir string) (files []string, err error) {
	if err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return
	}
	return
}

// ReadOsArgsParams reads the os.Args and maybe loads from the params file
func ReadOsArgsParams() ([]string, error) {
	args := os.Args[1:]
	if len(args) == 1 && strings.HasPrefix(args[0], "@") {
		paramsFile := args[0][1:]
		var err error
		args, err = readParamsFile(paramsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read params file %s: %v", paramsFile, err)
		}
	}
	return args, nil
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
