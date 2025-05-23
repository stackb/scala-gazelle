package collections

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile is a convenience function to copy file A to B.
func CopyFile(srcPath, dstPath string) error {
	// Open the source file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Create the destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Ensure all data is written to the destination
	err = dst.Sync()
	if err != nil {
		return err
	}

	return nil
}

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

// ReadArgsParamsFile reads the and maybe loads from the params file if the sole
// argument starts with '@'; if so args are loaded from that file.
func ReadArgsParamsFile(args []string) ([]string, error) {
	if len(args) != 1 {
		return args, nil
	}
	if !strings.HasPrefix(args[0], "@") {
		return args, nil
	}

	paramsFile := args[0][1:]
	var err error
	args, err = readParamsFile(paramsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read params file %s: %v", paramsFile, err)
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
		params = append(params, trimQuotes(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return params, nil
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
