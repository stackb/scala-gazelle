package parser

import (
	_ "embed"
)

//go:embed scalameta_parser.mjs
var parserrMjs string

//go:embed node_modules/scalameta-parsers/index.js
var scalametaParsersIndexJs string

//go:embed node.exe
var nodeExe []byte
