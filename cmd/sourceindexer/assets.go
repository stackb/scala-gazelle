package main

import (
	_ "embed"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

//go:embed sourceindexer.js
var sourceindexerJs string

//go:embed node_modules/scalameta-parsers/index.js
var scalametaParsersIndexJs string

//go:embed node.exe
var nodeExe []byte

func mustRestore(tmpDir string, opts *Options) {

}

// // mustRestore - must Restore assets or die.
// func mustRestore(opts *Options, baseDir string) {
// 	// unpack variable is provided by the go_embed data and is a
// 	// map[string][]byte such as {"/usr/share/games/fortune/literature.dat":
// 	// bytes... }
// 	for rel, bytes := range assets {
// 		filename := filepath.Join(baseDir, rel)
// 		dirname := filepath.Dir(filename)
// 		// log.Printf("file %s, dir %s, rel %d, abs %s, absdir: %s", file, dir, rel, abs, absdir)
// 		if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
// 			log.Fatalf("Failed to create asset dir %s: %v", dirname, err)
// 		}
// 		if err := ioutil.WriteFile(filename, bytes, os.ModePerm); err != nil {
// 			log.Fatalf("Failed to write asset %s: %v", filename, err)
// 		}

// 		switch filepath.Base(rel) {
// 		case "sourceindexer.js":
// 			opts.ScriptPath = filename
// 		case "node":
// 			opts.NodeBinPath = filename
// 		case "package.json":
// 			// cmd/sourceindexer/node_modules/scalameta-parsers/package.json -> cmd/sourceindexer/
// 			opts.NodePath = filepath.Dir(filepath.Dir(dirname))
// 		}

// 		if opts.Debug {
// 			log.Printf("Restored %s", filename)
// 		}
// 	}
// }

// writeFile - must Restore assets or die.
func mustWriteFile(baseDir, relDir string, data []byte) string {
	filename := filepath.Join(baseDir, relDir)
	dirname := filepath.Dir(filename)
	if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
		log.Fatalf("failed to create asset dir %s: %v", dirname, err)
	}
	if err := ioutil.WriteFile(filename, data, os.ModePerm); err != nil {
		log.Fatalf("failed to write asset %s: %v", filename, err)
	}
	return filename
}
