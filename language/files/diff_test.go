package files

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestDiffGeneratesPackageFilegroup(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path:    "BUILD.bazel",
			Content: ``,
		},
		{
			Path:    "hello.go",
			Content: `package hello`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if result, err := testutil.RunGazelle(t, dir, nil, "-lang=files", "-mode=diff", "-patch=p"); err != nil {
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected error: %v\n%+v", err, result)
		}
	}

	want := append(files, testtools.FileSpec{
		Path: "p",
		Content: `
--- BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -0,0 +1,12 @@
+load("@build_stack_scala_gazelle//rules:package_filegroup.bzl", "package_filegroup")
+
+package_filegroup(
+    name = "filegroup",
+    srcs = [
+        "BUILD.bazel",
+        "WORKSPACE",
+        "hello.go",
+    ],
+    visibility = ["//visibility:public"],
+)
+
`,
	})
	testtools.CheckFiles(t, dir, want)
}

func TestDiffUpdatesPackageFilegroup(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "BUILD.bazel",
			Content: `
load("@build_stack_scala_gazelle//rules:package_filegroup.bzl", "package_filegroup")

package_filegroup(
    name = "filegroup",
    srcs = [
        "BUILD.bazel",
        "WORKSPACE",
    ],
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path:    "hello.go",
			Content: `package hello`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if result, err := testutil.RunGazelle(t, dir, nil, "-lang=files", "-mode=diff", "-patch=p"); err != nil {
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected error: %v\n%+v", err, result)
		}
	}

	want := append(files, testtools.FileSpec{
		Path: "p",
		Content: `
--- BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -1,4 +1,3 @@
-
 load("@build_stack_scala_gazelle//rules:package_filegroup.bzl", "package_filegroup")
 
 package_filegroup(
@@ -6,6 +5,7 @@
     srcs = [
         "BUILD.bazel",
         "WORKSPACE",
+        "hello.go",
     ],
     visibility = ["//visibility:public"],
 )
`,
	})
	testtools.CheckFiles(t, dir, want)
}

func TestDiffGenetatesPackageFilegroupDeps(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path:    "a/BUILD.bazel",
			Content: ``,
		},
		{
			Path:    "b/BUILD.bazel",
			Content: ``,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if result, err := testutil.RunGazelle(t, dir, nil, "-lang=files", "-mode=diff", "-patch=p"); err != nil {
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected error: %v\n%+v", err, result)
		}
	}

	want := append(files, testtools.FileSpec{
		Path: "p",
		Content: `
--- a/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ a/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -0,0 +1,8 @@
+load("@build_stack_scala_gazelle//rules:package_filegroup.bzl", "package_filegroup")
+
+package_filegroup(
+    name = "filegroup",
+    srcs = ["BUILD.bazel"],
+    visibility = ["//visibility:public"],
+)
+
--- b/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ b/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -0,0 +1,8 @@
+load("@build_stack_scala_gazelle//rules:package_filegroup.bzl", "package_filegroup")
+
+package_filegroup(
+    name = "filegroup",
+    srcs = ["BUILD.bazel"],
+    visibility = ["//visibility:public"],
+)
+
--- /dev/null	1970-01-01 00:00:00.000000000 +0000
+++ BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -0,0 +1,12 @@
+load("@build_stack_scala_gazelle//rules:package_filegroup.bzl", "package_filegroup")
+
+package_filegroup(
+    name = "filegroup",
+    srcs = ["WORKSPACE"],
+    visibility = ["//visibility:public"],
+    deps = [
+        "//a:filegroup",
+        "//b:filegroup",
+    ],
+)
+
`,
	})
	testtools.CheckFiles(t, dir, want)
}
