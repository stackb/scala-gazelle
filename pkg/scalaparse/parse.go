package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

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
	Label string       `json:"label"`
	Srcs  []SourceFile `json:"srcs"`
}

// Parse runs the embedded parser
func Parse(label string, files []string) (*ParseResult, int, error) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		return nil, -1, err
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "sourceindexer.js")
	parserPath := filepath.Join(tmpDir, "node_modules", "scalameta-parsers", "index.js")

	if err := os.MkdirAll(filepath.Dir(parserPath), os.ModePerm); err != nil {
		return nil, -1, err
	}
	if err := ioutil.WriteFile(scriptPath, []byte(sourceindexerJs), os.ModePerm); err != nil {
		return nil, -1, err
	}
	if err := ioutil.WriteFile(parserPath, []byte(scalametaParsersIndexJs), os.ModePerm); err != nil {
		return nil, -1, err
	}

	args := append([]string{
		"./sourceindexer.js",
		"-l", label,
	}, files...)

	env := []string{
		"NODE_PATH=" + tmpDir,
	}
	if false { // TODO: pass options, conditionally add this
		env = append(env, "NODE_OPTIONS=--inspect-brk")
	}
	var stdout, stderr bytes.Buffer

	log.Println("args:", args)
	listFiles(tmpDir)
	exitCode, err := ExecNode(tmpDir, args, env, os.Stdin, &stdout, &stderr)
	if err != nil {
		return nil, exitCode, err
	}
	if exitCode != 0 {
		return nil, exitCode, fmt.Errorf(stderr.String())
	}

	log.Println("stdout:", stdout.String())

	var result ParseResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, exitCode, err
	}
	return &result, 0, nil
}
