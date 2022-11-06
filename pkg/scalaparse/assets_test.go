package scalaparse

import (
	"testing"
)

func TestEmbed(t *testing.T) {
	if len(sourceindexerMjs) == 0 {
		t.Error("embedded sourceindexer.mjs script is missing")
	}
	if len(scalametaParsersIndexJs) == 0 {
		t.Error("embedded node_modules/scalameta-parsers/index.js script is missing")
	}
	if len(nodeExe) == 0 {
		t.Errorf("embedded @build_bazel_rules_nodejs//toolchains/node:node_bin is missing: len %d", len(nodeExe))
	}
}
