package main

import (
	"flag"
	"path/filepath"
)

// Options are typically parsed from the command line.
type Options struct {
	// Debug enabled extra debugging
	Debug bool
	// RestoreEmbeddedFiles unpacks the bundled node interpreter and files.
	RestoreEmbeddedFiles bool
	// NodeBinPath is the path the node interpreter binary.
	NodeBinPath string
	// NodePath is the value for NODE_PATH env var for the node process
	NodePath string
	// ScriptPath is the path to the go_binary tool
	ScriptPath string
	// Files is a list of .scala source files to parse
	Files []string
}

// ParseOptions takes a temporary directory where embedded files should be written to,
// the path to the sourceindexer binary, and a list of command line arguments.
func ParseOptions(tmpDir, toolPath string, args []string) (*Options, error) {
	var opts Options

	var fs flag.FlagSet
	fs.StringVar(&opts.NodeBinPath, "node_bin_path", "", "override absolute location of node binary")
	fs.StringVar(&opts.NodePath, "node_path", "", "override absolute location of node modules")
	fs.StringVar(&opts.ScriptPath, "script_path", "", "override absolute location of sourceindexer script")
	fs.BoolVar(&opts.RestoreEmbeddedFiles, "embedded", true, "restore embedded files")
	fs.BoolVar(&opts.Debug, "debug", false, "debug mode")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	opts.Files = fs.Args()

	if opts.RestoreEmbeddedFiles {
		// mustRestore(&opts, tmpDir, embeddedInterpreter)
		// mustRestore(&opts, tmpDir, embeddedAssets)
		if opts.NodeBinPath == "" {
			opts.NodeBinPath = filepath.Join(tmpDir, "external/nodejs_darwin_amd64/bin/nodejs/bin/node")
		}
		if opts.NodePath == "" {
			opts.NodePath = filepath.Join(tmpDir, "pkg/scalaparse")
		}
		if opts.ScriptPath == "" {
			opts.ScriptPath = filepath.Join(tmpDir, "pkg/scalaparse/sourceindexer.js")
		}
	}

	// in the case where this tool is being run as an action the paths are
	// relative to the runfiles dir.
	runfilesDir := toolPath + ".runfiles"
	if opts.NodeBinPath == "" {
		opts.NodeBinPath = filepath.Join(runfilesDir, "nodejs_darwin_amd64/bin/nodejs/bin/node")
	}
	if opts.NodePath == "" {
		opts.NodePath = filepath.Join(runfilesDir, "scala_gazelle/pkg/scalaparse")
	}
	if opts.ScriptPath == "" {
		opts.ScriptPath = filepath.Join(runfilesDir, "scala_gazelle/pkg/scalaparse/sourceindexer.js")
	}

	return &opts, nil
}
