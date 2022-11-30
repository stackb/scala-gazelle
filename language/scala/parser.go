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
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/scalaparse"
)

// ref: https://raw.githubusercontent.com/bazelbuild/rules_python/main/gazelle/parser.go

// scalaSourceParser implements a parser frontend for scala files that extracts
// the index information.  The parser backend runs as a separate process.
type scalaSourceParser struct {
	parserToolPath string
	parserStdin    io.Writer
	parserStdout   io.Reader
	parserCancel   func()
	parserMutex    sync.Mutex
}

func (p *scalaSourceParser) start() error {
	parseTool, err := bazel.Runfile(p.parserToolPath)
	if err != nil {
		scalaparse.ListFiles(".")
		log.Fatalf("failed to initialize parser: %v\n", err)
		return err
	}

	ctx := context.Background()
	ctx, parserCancel := context.WithTimeout(ctx, time.Minute*5)
	cmd := exec.CommandContext(ctx, parseTool, "-embedded")
	p.parserCancel = parserCancel

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		os.Exit(1)
	}
	p.parserStdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		return err
	}
	p.parserStdout = stdout

	if err := cmd.Start(); err != nil {
		log.Printf("failed to initialize parser: %v\n", err)
		return err
	}

	return nil
}

func (p *scalaSourceParser) stop() {
	if p.parserCancel != nil {
		p.parserCancel()
	}
}

// parseAll parses all provided Scala files by consecutively calling p.parse.
func (p *scalaSourceParser) parseAll(filenames []string) ([]*sppb.File, error) {
	files := make([]*sppb.File, len(filenames))
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
func (p *scalaSourceParser) parse(filename string) (*sppb.File, error) {
	p.parserMutex.Lock()
	defer p.parserMutex.Unlock()

	fmt.Fprintln(p.parserStdin, filename)
	reader := bufio.NewReader(p.parserStdout)
	data, err := reader.ReadBytes(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	data = data[:len(data)-1]
	var spec sppb.File
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	return &spec, nil
}
