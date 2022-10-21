package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetPrefix("sourceindexer.go: ")
	log.SetFlags(0) // don't print timestamps

	tmpDir := os.TempDir()

	opts, err := ParseOptions(tmpDir, os.Args[0], os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if opts.Debug {
		// log.Println(os.Environ())
		listFiles(".")
	}

	// args := append([]string{opts.ScriptPath}, opts.Files...)
	// env := []string{"NODE_PATH=" + opts.NodePath}

	// exitCode, err := run(opts.NodeBinPath, args, ".", env)
	// if err != nil {
	// 	log.Print(err)
	// }
	// os.Exit(exitCode)
}

// listFiles - convenience debugging function to log the files under a given dir
func listFiles(dir string) error {
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
