package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

// debugParse is a debug flag for use by a developer
const debugParse = false

type SourceFile struct {
	Filename string `json:"filename,omitempty"`

	Classes  []string `json:"classes,omitempty"`
	Imports  []string `json:"imports,omitempty"`
	Names    []string `json:"names,omitempty"`
	Objects  []string `json:"objects,omitempty"`
	Packages []string `json:"packages,omitempty"`
	Traits   []string `json:"traits,omitempty"`
	Types    []string `json:"types,omitempty"`
	Error    string   `json:"error,omitempty"`

	Extends map[string][]string `json:"extends,omitempty"`
}

type ParseResult struct {
	Label    string       `json:"label"`
	Srcs     []SourceFile `json:"srcs"`
	Stdout   string
	Stderr   string
	ExitCode int
}

// Parse runs the embedded parser in batch mode.
func Parse(label string, files []string) (*ParseResult, error) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "sourceindexer.js")
	parserPath := filepath.Join(tmpDir, "node_modules", "scalameta-parsers", "index.js")

	if err := os.MkdirAll(filepath.Dir(parserPath), os.ModePerm); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(scriptPath, []byte(sourceindexerJs), os.ModePerm); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(parserPath, []byte(scalametaParsersIndexJs), os.ModePerm); err != nil {
		return nil, err
	}

	if debugParse {
		listFiles(".")
	}

	args := append([]string{
		"./sourceindexer.js",
		"-l", label,
	}, files...)

	env := []string{
		"NODE_PATH=" + tmpDir,
	}

	var stdout, stderr bytes.Buffer

	exitCode, err := ExecNode(tmpDir, args, env, os.Stdin, &stdout, &stderr)

	result := &ParseResult{
		Stderr:   stderr.String(),
		Stdout:   stdout.String(),
		ExitCode: exitCode,
	}

	if err != nil || exitCode != 0 {
		return result, err
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, err
	}

	// after a successful json parse, unset the stdout, we don't care anymore
	result.Stdout = ""

	return result, nil
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
