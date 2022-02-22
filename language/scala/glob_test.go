package scala

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
)

// TestParseGlob tests the parsing of a starlark glob.
func TestParseGlob(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		want rule.GlobValue
	}{
		"empty glob": {
			text: `glob()`,
			want: rule.GlobValue{},
		},
		"default include list - empty": {
			text: `glob([])`,
			want: rule.GlobValue{},
		},
		"default include list - one pattern": {
			text: `glob(["A.scala"])`,
			want: rule.GlobValue{Patterns: []string{"A.scala"}},
		},
		"default include list - two patterns": {
			text: `glob(["A.scala", "B.scala"])`,
			want: rule.GlobValue{Patterns: []string{"A.scala", "B.scala"}},
		},
		"exclude list - single exclude": {
			text: `glob([], exclude=["C.scala"])`,
			want: rule.GlobValue{Excludes: []string{"C.scala"}},
		},
		"complex value - not supported": {
			text: `glob(get_include_list(), get_exclude_list())`,
			want: rule.GlobValue{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			content := fmt.Sprintf("test_rule(srcs = %s)", tc.text)
			file, err := build.Parse("BUILD", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			r := file.Rules("test_rule")[0]

			got := parseGlob(r.Attr("srcs").(*build.CallExpr))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseGlob (-want +got):\n%s", diff)
			}
		})
	}
}

// TestApplyGlob tests the application of a starlark glob over a filesystem.
func TestApplyGlob(t *testing.T) {
	for name, tc := range map[string]struct {
		glob  rule.GlobValue
		files []testtools.FileSpec
		want  []string
	}{
		"empty glob": {
			glob: rule.GlobValue{},
			files: []testtools.FileSpec{
				{Path: "src/A.scala"},
			},
			want: nil,
		},
		"single explicit match": {
			glob: rule.GlobValue{Patterns: []string{"src/A.scala"}},
			files: []testtools.FileSpec{
				{Path: "src/A.scala"},
			},
			want: []string{"src/A.scala"},
		},
		"doublestar match": {
			glob: rule.GlobValue{Patterns: []string{"**/*.scala"}},
			files: []testtools.FileSpec{
				{Path: "src/A.scala"},
				{Path: "src/B.scala"},
			},
			want: []string{"src/A.scala", "src/B.scala"},
		},
		"doublestar match + exclude": {
			glob: rule.GlobValue{Patterns: []string{"test/**/*.scala"}, Excludes: []string{"test/**/Manual*.scala"}},
			files: []testtools.FileSpec{
				{Path: "test/A.scala"},
				{Path: "test/B.scala"},
				{Path: "test/ManualXXX.scala"},
			},
			want: []string{"test/A.scala", "test/B.scala"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, err := bazel.NewTmpDir("")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			for _, file := range tc.files {
				abs := filepath.Join(tmpDir, file.Path)
				dir := filepath.Dir(abs)
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					t.Fatal(err)
				}
				if !file.NotExist {
					if err := ioutil.WriteFile(abs, []byte(file.Content), os.ModePerm); err != nil {
						t.Fatal(err)
					}
				}
			}
			got := applyGlob(tc.glob, os.DirFS(tmpDir))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("applyGlob (-want +got):\n%s", diff)
			}
		})
	}
}
