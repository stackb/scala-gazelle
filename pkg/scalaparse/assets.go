package scalaparse

import (
	_ "embed"
)

//go:embed scalaparser.mjs
var scalaparserMjs string

//go:embed node_modules/scalameta-parsers/index.js
var scalametaParsersIndexJs string

//go:embed node.exe
var nodeExe []byte
