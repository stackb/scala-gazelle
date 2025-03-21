package files

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"

	"github.com/stackb/scala-gazelle/pkg/collections"
)

func TestDiffExisting(t *testing.T) {

	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "BUILD.bazel",
			Content: `
# gazelle:prefix example.com/hello
`,
		},
		{
			Path:    "hello.go",
			Content: `package hello`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	collections.ListFiles(dir)

	wantError := "encountered changes while running diff"
	if output, err := RunGazelle(t, dir, []string{"-mode=diff", "-patch=p"}); err != nil {
		if err.Error() != wantError {
			t.Fatalf("got %q; want %q", err, wantError)
		}
	} else {
		t.Fatal("expected a diff", output)
	}

	want := append(files, testtools.FileSpec{
		Path: "p",
		Content: `
--- BUILD.bazel	1970-01-01 00:00:00.000000001 +0000
+++ BUILD.bazel	1970-01-01 00:00:00.000000001 +0000
@@ -1,2 +1,10 @@
+load("@io_bazel_rules_go//go:def.bzl", "go_library")
 
 # gazelle:prefix example.com/hello
+
+go_library(
+    name = "hello",
+    srcs = ["hello.go"],
+    importpath = "example.com/hello",
+    visibility = ["//visibility:public"],
+)
`,
	})
	testtools.CheckFiles(t, dir, want)
}
