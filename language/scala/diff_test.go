package scala

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestDiff(t *testing.T) {
	for name, tc := range map[string]struct {
		files        []testtools.FileSpec
		args         []string
		env          []string
		checkErr     bool
		wantErr      string
		checkStdout  bool
		wantStdout   string
		checkStderr  bool
		wantStderr   string
		wantExitCode int
		want         string
	}{
		"degenerate": {
			files: []testtools.FileSpec{
				{Path: "BUILD.bazel"},
			},
		},
		"known scale_rules do not require pre-registration of an implementating provider": {
			files: []testtools.FileSpec{
				{
					Path: "BUILD.bazel",
					Content: `
# gazelle:scala_rule scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary
`,
				},
			},
			env: []string{
				"SCALA_GAZELLE_SHOW_PROGRESS=0",
				"SCALA_GAZELLE_SHOW_COVERAGE=0",
				"SCALA_GAZELLE_LOG_FILE=/tmp/scala-gazelle.log",
				"SCALA_GAZELLE_ANNOUNCE_LOG_FILE=0",
			},
			wantExitCode: 0,
			checkStderr:  true,
			wantStderr:   ``,
			wantErr:      ``,
		},

		// "custom scale_rule requires pre-registration of an implementating provider": {
		// 	files: []testtools.FileSpec{
		// 		{
		// 			Path:    "BUILD.bazel",
		// 			Content: `# gazelle:scala_rule scala_helper_library implementation //rules:scala.bzl%scala_helper_library`,
		// 		},
		// 	},
		// 	checkStderr: true,
		// 	wantStderr:  `gazelle: rule not registered: "//rules:scala.bzl%scala_helper_library" (available: [@build_stack_scala_gazelle//rules:scala_files.bzl%scala_files @build_stack_scala_gazelle//rules:scala_files.bzl%scala_fileset @build_stack_scala_gazelle//rules:semanticdb_index.bzl%semanticdb_index @io_bazel_rules_scala//scala:scala.bzl%scala_binary @io_bazel_rules_scala//scala:scala.bzl%scala_library @io_bazel_rules_scala//scala:scala.bzl%scala_macro_library @io_bazel_rules_scala//scala:scala.bzl%scala_test])`,
		// },
	} {
		t.Run(name, func(t *testing.T) {
			files := append(tc.files, testtools.FileSpec{Path: "WORKSPACE"})
			args := append([]string{"-lang=scala"}, tc.args...)
			if !tc.checkStderr {
				args = append(args, "-mode=diff", "-patch=p")
			}

			dir, cleanup := testtools.CreateFiles(t, files)
			env := append(tc.env, fmt.Sprintf("SCALA_GAZELLE_LOG_FILE=%s/scala-gazelle.log", dir))

			defer cleanup()

			stdout, stderr, exitCode, err := testutil.RunGazelle(t, dir, env, args...)

			if diff := cmp.Diff(tc.wantExitCode, exitCode); diff != "" {
				t.Errorf("exit code (-want +got):\n%s", diff)
			}

			if tc.checkStdout {
				if diff := cmp.Diff(tc.wantStdout, strings.TrimSpace(stdout)); diff != "" {
					t.Errorf("stdout (-want +got):\n%s", diff)
				}
			}
			if tc.checkStderr {
				var gotErr string
				if err != nil {
					gotErr = err.Error()
				}
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("err (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tc.wantStderr, strings.TrimSpace(stderr)); diff != "" {
					t.Errorf("stderr (-want +got):\n%s", diff)
				}
			} else {
				testtools.CheckFiles(t, dir, append(files, testtools.FileSpec{
					Path:    "p",
					Content: tc.want,
				}))
			}
		})
	}
}

// func SkipTestExistingRuleRegistration(t *testing.T) {
// 	files := []testtools.FileSpec{
// 		{
// 			Path: "WORKSPACE",
// 		},
// 		{
// 			Path: "BUILD.bazel",
// 			Content: `
// # gazelle:scala_rule scala_helper_library implementation //rules:scala.bzl%scala_helper_library
// `,
// 		},
// 	}
// 	dir, cleanup := testtools.CreateFiles(t, files)
// 	defer cleanup()

// 	stdout, stderr, exitCode, err := testutil.RunGazelle(t, dir, nil,
// 		"-lang=scala", "-mode=diff", "-patch=p",
// 		"-strict",
// 		"-existing_scala_library_rule=//rules:scala.bzl%scala_helper_library",
// 	)
// 	t.Logf("EXIT CODE: %d", exitCode)
// 	t.Logf("ERR: %v", err)
// 	t.Logf("STDOUT:\n%s", stdout)
// 	t.Logf("STDERR:\n%s", stderr)

// 	want := ``
// 	got := strings.TrimSpace(stderr)

// 	if diff := cmp.Diff(want, got); diff != "" {
// 		t.Errorf("stderr (-want +got):\n%s", diff)
// 	}
// }

// func SkipTestDiffSourceProvider(t *testing.T) {
// 	files := []testtools.FileSpec{
// 		{
// 			Path: "WORKSPACE",
// 		},
// 		{
// 			Path: "BUILD.bazel",
// 			Content: `load("@io_bazel_rules_scala//scala:scala.bzl", "scala_binary")

// # gazelle:scala_rule scala_helper_library implementation //rules:scala.bzl%scala_helper_library
// # gazelle:scala_rule scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary
// # gazelle:resolve_kind_rewrite_name scala_helper_library %{name} %{name}_helper

// scala_binary(
//     name = "app",
//     srcs = ["App.scala"],
//     main_class = "app.App",
//     deps = [
//         "//lib:lib_helper",  # DIRECT
//     ],
// )
// `,
// 		},
// 		{Path: "App.scala",
// 			Content: `
// package app

// object App {
//   def main(args: Array[String]): Unit = {}
// }
// `,
// 		},
// 		{Path: "lib/BUILD.bazel",
// 			Content: `load("//rules:scala.bzl", "scala_helper_library")

// scala_helper_library(
//     name = "lib",
//     srcs = ["Helper.scala"],
// )
// `,
// 		},
// 		{Path: "lib/Helper.scala",
// 			Content: `
// package lib

// object Helper {
// }
// `,
// 		},
// 	}
// 	dir, cleanup := testtools.CreateFiles(t, files)
// 	if true {
// 		defer cleanup()
// 	}

// 	stdout, stderr, err := testutil.RunGazelle(t, dir, nil,
// 		"-lang=scala", "-mode=diff", "-patch=p",
// 		"-strict",
// 		"-existing_scala_library_rule=@io_bazel_rules_scala//scala:scala.bzl%_scala_library",
// 		"-scala_symbol_provider=source",
// 	)
// 	t.Logf("STDOUT:\n%s", stdout)
// 	t.Logf("STDERR:\n%s", stderr)
// 	collections.ListFiles(dir)

// 	if err != nil {
// 		if err.Error() != "exit status 1" {
// 			t.Fatalf("unexpected error: %v\nSTDOUT:\n%s\nSTDERR:\n%s", err, stdout, stderr)
// 		}
// 	}

// 	want := append(files, testtools.FileSpec{
// 		Path: "p",
// 		Content: `
// xxx
// 		`,
// 	})
// 	testtools.CheckFiles(t, dir, want)
// }
