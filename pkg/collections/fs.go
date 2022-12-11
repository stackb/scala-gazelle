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
