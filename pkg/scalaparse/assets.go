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
