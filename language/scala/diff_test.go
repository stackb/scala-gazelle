package scala

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

type testCase struct {
	files        []testtools.FileSpec
	args         []string
	env          []string
	wantErr      string
	wantStdout   string
	wantStderr   string
	wantExitCode int
	wantDiff     string
	skipCleanup  bool
}

func (tc *testCase) Run(name string, t *testing.T) {
	t.Run(name, func(t *testing.T) {
		files := append(tc.files, testtools.FileSpec{Path: "WORKSPACE"})
		args := append([]string{"-lang=scala"}, tc.args...)
		if tc.wantDiff != "" {
			args = append(args, "-mode=diff", "-patch=p")
		}

		dir, cleanup := testtools.CreateFiles(t, files)
		env := append(tc.env,
			"SCALA_GAZELLE_SHOW_PROGRESS=0",
			"SCALA_GAZELLE_SHOW_COVERAGE=0",
			fmt.Sprintf("SCALA_GAZELLE_LOG_FILE=%s/scala-gazelle.log", dir),
		)
		if !tc.skipCleanup {
			defer cleanup()
		}

		var gotErr string
		result, err := testutil.RunGazelle(t, dir, env, args...)
		if err != nil {
			gotErr = err.Error()
			if tc.wantDiff == "" {
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("err (-want +got):\n%s", diff)
				}
			}
		}
		if result == nil {
			return
		}

		if tc.wantDiff == "" {
			if diff := cmp.Diff(tc.wantExitCode, result.ExitCode); diff != "" {
				t.Errorf("exit code (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantStdout, strings.TrimSpace(result.Stdout)); diff != "" {
				t.Errorf("stdout (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantStderr, strings.TrimSpace(result.Stderr)); diff != "" {
				t.Errorf("stderr (-want +got):\n%s", diff)
			}
		} else {
			files = append(files, testtools.FileSpec{
				Path:    "p",
				Content: tc.wantDiff,
			})
			testutil.ListFiles(t, dir)
			testtools.CheckFiles(t, dir, files)
		}

	})
}

func TestDiffDegenerate(t *testing.T) {
	// in the degenerate case, there is no scala-gazelle content in the
	// BUILD file, and no diff is expected.
	for name, tc := range map[string]*testCase{
		"empty build file": {
			files: []testtools.FileSpec{
				{
					Path: "BUILD.bazel",
				},
			},
		},
	} {
		tc.Run(name, t)
	}
}

// some scala rules come configured out of the box.  These don't need a flag to
// associate the load%kind with the existing_scala_rule.go implementation.  If
// unknown, it prints out the known rules.
func TestDiffKnownScalaRuleRegistration(t *testing.T) {
	testCases := make(map[string]*testCase)

	for _, kind := range []string{
		"scala_library",
		"scala_binary",
		"scala_macro_library",
		"scala_test",
	} {
		testCases[kind] = &testCase{
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: "# gazelle:scala_rule " + kind + " implementation @io_bazel_rules_scala//scala:scala.bzl%" + kind,
				},
			},
		}
	}

	for _, kind := range []string{
		"scala_foo",
	} {
		testCases[kind] = &testCase{
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: "# gazelle:scala_rule " + kind + " implementation @io_bazel_rules_scala//scala:scala.bzl%" + kind,
				},
			},
			wantExitCode: 1,
			wantErr:      "exit status 1",
			wantStderr:   `gazelle: rule not registered: "@io_bazel_rules_scala//scala:scala.bzl%scala_foo" (available: [@build_stack_scala_gazelle//rules:scala_files.bzl%scala_files @build_stack_scala_gazelle//rules:scala_files.bzl%scala_fileset @build_stack_scala_gazelle//rules:semanticdb_index.bzl%semanticdb_index @io_bazel_rules_scala//scala:scala.bzl%scala_binary @io_bazel_rules_scala//scala:scala.bzl%scala_library @io_bazel_rules_scala//scala:scala.bzl%scala_macro_library @io_bazel_rules_scala//scala:scala.bzl%scala_test])`,
		}
	}

	for name, tc := range testCases {
		tc.Run(name, t)
	}
}

// custom scala rules need a flag to declare the type of the
// existing_scala_rule.go implementation.  These come in binary|library|test
// "flavors".  Where there is a flag for each existing rule flavor.
func TestDiffCustomScalaRuleRegistration(t *testing.T) {
	testCases := make(map[string]*testCase)

	for flavor, kind := range map[string]string{
		"library": "my_scala_library",
		"binary":  "my_scala_binary",
		"test":    "my_scala_test",
	} {
		testCases[kind] = &testCase{
			args: []string{"--existing_scala_" + flavor + "_rule=@//scala:scala.bzl%" + kind},
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: "# gazelle:scala_rule " + kind + " implementation @//scala:scala.bzl%" + kind,
				},
			},
		}
	}

	for flavor, kind := range map[string]string{
		"macro": "my_scala_macro", // this flavor does not exist, macro rules aren't different per-se
	} {
		testCases[kind] = &testCase{
			args: []string{"--existing_scala_" + flavor + "_rule=@//scala:scala.bzl%" + kind},
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: "# gazelle:scala_rule " + kind + " implementation @//scala:scala.bzl%" + kind,
				},
			},
			wantErr:      "exit status 1",
			wantExitCode: 1,
			wantStderr:   "flag provided but not defined: -existing_scala_macro_rule\ngazelle: Try -help for more information.",
		}
	}

	for name, tc := range testCases {
		tc.Run(name, t)
	}
}

// symbol providers need to be registered to be enabled.  This group of tests
// names the known "out of the box" providers .
func TestKnownSymbolProviders(t *testing.T) {
	testCases := make(map[string]*testCase)

	for _, name := range []string{
		"source",
		"semanticdb",
		"protobuf",
		"java",
		"maven",
	} {
		testCases[name] = &testCase{
			args:         []string{"--scala_symbol_provider=" + name},
			wantExitCode: 0,
		}
	}

	for _, name := range []string{
		"foo",
	} {
		testCases[name] = &testCase{
			args:         []string{"--scala_symbol_provider=" + name},
			wantStderr:   fmt.Sprintf("gazelle: resolver.SymbolProvider not found: %q", name),
			wantExitCode: 1,
			wantErr:      "exit status 1",
		}
	}

	for name, tc := range testCases {
		tc.Run(name, t)
	}
}

// conflict resolvers need to be registered, and declared to be enable or not in
// a BUILD file.  This group of tests names the known "out of the box" conflict
// resolvers.
func TestKnownConflictResolvers(t *testing.T) {
	testCases := make(map[string]*testCase)

	for _, name := range []string{
		// helps decide which dependency to take when a symbol is provided by both
		// proto_scala_library and grpc_scala_library rules
		"scala_proto_package",
		// helps decide which dependency to take when a symbol is provided by both
		// grpc_scala_library and grpc_zio_scala_library rules.
		// "-scala_conflict_resolver=scala_grpc_zio",
		"scala_grpc_zio",
		// resolves conflicts in favor of those provided by the platform (e.g
		// 'java.lang.String' or 'scala.sys').
		"predefined_label",
		// resolves package conflicts using the java_index.preferred_deps attribute.
		// "-scala_conflict_resolver=preferred_deps",
		"preferred_deps",
	} {
		testCases[name] = &testCase{
			args: []string{"--scala_conflict_resolver=" + name},
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: "# gazelle:resolve_conflicts +" + name, // intent modifier (+/-) is optional
				},
			},

			wantExitCode: 0,
		}
	}

	for _, name := range []string{
		"foo",
	} {
		testCases[name] = &testCase{
			args:         []string{"--scala_conflict_resolver=" + name},
			wantStderr:   fmt.Sprintf("gazelle: -scala_conflict_resolver not found: %q", name),
			wantExitCode: 1,
			wantErr:      "exit status 1",
		}
	}

	for name, tc := range testCases {
		tc.Run(name, t)
	}
}

// deps cleaners need to be registered, and declared to be enable or not in a
// BUILD file.  This group of tests names the known "out of the box" dependency
// cleaners.
func TestKnownDepsCleaners(t *testing.T) {
	testCases := make(map[string]*testCase)

	for _, name := range []string{
		// removes duplicate/overlapping proto/grpc deps from scala rules.
		// NOTE: this implementation is in a client repo
		// "scala_proto_grpc_deps_cleaner",
	} {
		testCases[name] = &testCase{
			args: []string{"--scala_deps_cleaner=" + name},
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: "# gazelle:scala_deps_cleaner +" + name, // intent modifier (+/-) is optional
				},
			},

			wantExitCode: 0,
		}
	}

	for _, name := range []string{
		"foo",
	} {
		testCases[name] = &testCase{
			args:         []string{"--scala_deps_cleaner=" + name},
			wantStderr:   fmt.Sprintf("gazelle: -scala_deps_cleaner not found: %q", name),
			wantExitCode: 1,
			wantErr:      "exit status 1",
		}
	}

	for name, tc := range testCases {
		tc.Run(name, t)
	}
}

// This group of tests demonstrates that an existing scala rule is found and
// parsed by the source provider.
func TestSourceProvider(t *testing.T) {
	testCases := make(map[string]*testCase)
	defaultArgs := []string{
		"-scala_symbol_provider=source",
	}
	defaultEnv := []string{}
	defaultFiles := []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `# gazelle:scala_rule scala_library implementation @io_bazel_rules_scala//scala:scala.bzl%scala_library
# gazelle:scala_rule scala_library enabled true
`,
		},
	}

	testCases["adds direct dependency (same package)"] = &testCase{
		args: defaultArgs,
		env:  defaultEnv,
		files: append(defaultFiles, []testtools.FileSpec{
			{
				Path: "lib/BUILD.bazel",
				Content: `load("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")

scala_library(
    name = "a",
    srcs = ["A.scala"],
)

scala_library(
    name = "b",
    srcs = ["B.scala"],
)
`,
			},
			{
				Path: "lib/A.scala",
				Content: `
package lib

object A {}
`,
			},
			{
				Path: "lib/B.scala",
				Content: `
package lib

import lib.A

object B {}
`,
			},
		}...),
		wantDiff: `
--- lib/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ lib/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -8,5 +8,8 @@
 scala_library(
     name = "b",
     srcs = ["B.scala"],
+    deps = [
+        ":a",  # DIRECT
+    ],
 )
`,
	}

	testCases["adds direct dependency (cross package)"] = &testCase{
		args: defaultArgs,
		env:  defaultEnv,
		files: append(defaultFiles, []testtools.FileSpec{
			{
				Path: "a/BUILD.bazel",
				Content: `load("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")

scala_library(
    name = "a",
    srcs = ["A.scala"],
)
`,
			},
			{
				Path: "a/A.scala",
				Content: `
package a

object A {}
`,
			},
			{
				Path: "b/BUILD.bazel",
				Content: `load("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")

scala_library(
    name = "b",
    srcs = ["B.scala"],
)
`,
			},
			{
				Path: "b/B.scala",
				Content: `
package b

import a.A

object B {}
`,
			},
		}...),
		wantDiff: `
--- b/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ b/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -3,5 +3,8 @@
 scala_library(
     name = "b",
     srcs = ["B.scala"],
+    deps = [
+        "//a",  # DIRECT
+    ],
 )
`,
	}

	testCases["adds extends dependency with exports (same package)"] = &testCase{
		args: append(defaultArgs),
		env:  defaultEnv,
		files: append(defaultFiles, []testtools.FileSpec{
			{
				Path: "lib/BUILD.bazel",
				Content: `load("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")

scala_library(
    name = "a",
    srcs = ["A.scala"],
)

scala_library(
    name = "b",
    srcs = ["B.scala"],
)
`,
			},
			{
				Path: "lib/A.scala",
				Content: `
package lib

object A {}
`,
			},
			{
				Path: "lib/B.scala",
				Content: `
package lib

import lib.A

object B extends A {}
`,
			},
		}...),
		wantDiff: `
--- lib/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ lib/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -8,5 +8,11 @@
 scala_library(
     name = "b",
     srcs = ["B.scala"],
+    exports = [
+        ":a",  # EXTENDS
+    ],
+    deps = [
+        ":a",  # DIRECT
+    ],
 )
`,
	}

	testCases["adds direct dependency from scala_fileset"] = &testCase{
		args: append(defaultArgs, "-scala_fileset_file=scala_fileset.json"),
		env:  append(defaultEnv, "SCALA_GAZELLE_ALLOW_RUNTIME_PARSING=false"),
		files: append(defaultFiles, []testtools.FileSpec{
			{
				Path: "scala_fileset.json",
				Content: `{
    "rules": [
        {
            "label": "@//lib:scala_files",
            "kind": "scala_files",
            "files": [
                {
                    "filename": "lib/A.scala",
                    "packages": [
                        "lib"
                    ],
                    "objects": [
                        "lib.A"
                    ]
                },
                {
                    "filename": "lib/B.scala",
                    "imports": [
                        "lib.A"
                    ],
                    "packages": [
                        "lib"
                    ],
                    "objects": [
                        "lib.B"
                    ]
                }
            ]
        }
    ]
}`,
			},
			{
				Path: "lib/BUILD.bazel",
				Content: `load("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")

scala_library(
    name = "a",
    srcs = ["A.scala"],
)

scala_library(
    name = "b",
    srcs = ["B.scala"],
)
`,
			},
			{
				Path: "lib/A.scala",
				Content: `
package lib

object A {}
`,
			},
			{
				Path: "lib/B.scala",
				Content: `
package lib

import lib.A

object B {}
`,
			},
		}...),
		// 		wantDiff: `
		// --- lib/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
		// +++ lib/BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
		// @@ -8,5 +8,11 @@
		//  scala_library(
		//      name = "b",
		//      srcs = ["B.scala"],
		// +    deps = [
		// +        ":a",  # DIRECT
		// +    ],
		//  )
		// `,
	}

	for name, tc := range testCases {
		tc.Run(name, t)
	}
}
