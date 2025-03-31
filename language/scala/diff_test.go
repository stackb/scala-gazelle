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
		wantErr      string
		wantStdout   string
		wantStderr   string
		wantExitCode int
		wantDiff     string
	}{
		"degenerate": {
			files: []testtools.FileSpec{
				{
					Path: "BUILD.bazel",
				},
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
		},
		"custom scale_rule requires pre-registration of an implementating provider": {
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: `# gazelle:scala_rule scala_helper_library implementation //rules:scala.bzl%scala_helper_library`,
				},
			},
			wantExitCode: 1,
			wantStderr:   `gazelle: rule not registered: "//rules:scala.bzl%scala_helper_library" (available: [@build_stack_scala_gazelle//rules:scala_files.bzl%scala_files @build_stack_scala_gazelle//rules:scala_files.bzl%scala_fileset @build_stack_scala_gazelle//rules:semanticdb_index.bzl%semanticdb_index @io_bazel_rules_scala//scala:scala.bzl%scala_binary @io_bazel_rules_scala//scala:scala.bzl%scala_library @io_bazel_rules_scala//scala:scala.bzl%scala_macro_library @io_bazel_rules_scala//scala:scala.bzl%scala_test])`,
			wantErr:      "exit status 1",
		},
		"custom scale_rule requires pre-registration of an implementating provider (fixed)": {
			files: []testtools.FileSpec{
				{
					Path:    "BUILD.bazel",
					Content: `# gazelle:scala_rule scala_helper_library implementation //rules:scala.bzl%scala_helper_library`,
				},
			},
			args: []string{"--existing_scala_library_rule=//rules:scala.bzl%scala_helper_library"},
		},
	} {
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

			defer cleanup()

			var gotErr string
			result, err := testutil.RunGazelle(t, dir, env, args...)
			if err != nil {
				gotErr = err.Error()
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("err (-want +got):\n%s", diff)
				}
			}
			if result == nil {
				return
			}

			if diff := cmp.Diff(tc.wantExitCode, result.ExitCode); diff != "" {
				t.Errorf("exit code (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantStdout, strings.TrimSpace(result.Stdout)); diff != "" {
				t.Errorf("stdout (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantStderr, strings.TrimSpace(result.Stderr)); diff != "" {
				t.Errorf("stderr (-want +got):\n%s", diff)
			}

			if tc.wantDiff != "" {
				files = append(files, testtools.FileSpec{
					Path:    "p",
					Content: tc.wantDiff,
				})
			}

			testtools.CheckFiles(t, dir, files)
		})
	}
}
