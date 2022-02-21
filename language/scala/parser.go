package scala

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// ref: https://raw.githubusercontent.com/bazelbuild/rules_python/main/gazelle/parser.go

var (
	parserStdin  io.Writer
	parserStdout io.Reader
	parserMutex  sync.Mutex
)

func init() {
	parseTool, err := bazel.Runfile("sourceindexer")
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		index.ListFiles(".")
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, parserCancel := context.WithTimeout(ctx, time.Minute*5)
	cmd := exec.CommandContext(ctx, parseTool, "-embedded")

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}
	parserStdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}
	parserStdout = stdout

	if err := cmd.Start(); err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}

	go func() {
		defer parserCancel()
		if err := cmd.Wait(); err != nil {
			log.Printf("failed to wait for parser: %v\n", err)
			os.Exit(1)
		}
	}()
}

// scalaSourceParser implements a parser for scala files that extracts the index information.
type scalaSourceParser struct{}

// newScalaSourceParser constructs a new scalaSourceParser.
func newScalaSourceParser() *scalaSourceParser {
	return &scalaSourceParser{}
}

// parseAll parses all provided Scala files by consecutively calling p.parse.
func (p *scalaSourceParser) parseAll(filenames []string) ([]*index.ScalaFileSpec, error) {
	files := make([]*index.ScalaFileSpec, len(filenames))
	for i, filename := range filenames {
		file, err := p.parse(filename)
		if err != nil {
			return nil, err
		}
		files[i] = file
	}
	return files, nil
}

// parse parses a Scala file and returns the index. An error is raised if
// communicating with the long-lived Scala parser over stdin and stdout fails.
func (p *scalaSourceParser) parse(filename string) (*index.ScalaFileSpec, error) {
	parserMutex.Lock()
	defer parserMutex.Unlock()

	fmt.Fprintln(parserStdin, filename)
	reader := bufio.NewReader(parserStdout)
	data, err := reader.ReadBytes(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	data = data[:len(data)-1]
	var spec index.ScalaFileSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	return &spec, nil
}
