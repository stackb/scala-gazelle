package collections

import (
	"log"
	"os"
	"path/filepath"
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
