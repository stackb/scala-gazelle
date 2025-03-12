package glob

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"

	"github.com/stackb/scala-gazelle/pkg/bazel"
)

// TestParseGlob tests the parsing of a starlark glob.
func TestParseGlob(t *testing.T) {
	for name, tc := range map[string]struct {
		// prelude is an optional chunk of BUILD file content
		prelude string
		// glob is the starlark text of the glob
		glob string
		// want is the expected parsed structure
		want rule.GlobValue
	}{
		"empty glob": {
			glob: `glob()`,
			want: rule.GlobValue{},
		},
		"default include list - empty": {
			glob: `glob([])`,
			want: rule.GlobValue{},
		},
		"default include list - one pattern": {
			glob: `glob(["A.scala"])`,
			want: rule.GlobValue{Patterns: []string{"A.scala"}},
		},
		"default include list - two patterns": {
			glob: `glob(["A.scala", "B.scala"])`,
			want: rule.GlobValue{Patterns: []string{"A.scala", "B.scala"}},
		},
		"exclude list - single exclude": {
			glob: `glob([], exclude=["C.scala"])`,
			want: rule.GlobValue{Excludes: []string{"C.scala"}},
		},
		"global pattern": {
			prelude: `LIST = ["A.scala"]`,
			glob:    `glob(LIST)`,
			want:    rule.GlobValue{Patterns: []string{"A.scala"}},
		},
		"global pattern and exclude": {
			prelude: `
INCLUDES = ["A.scala"]
EXCLUDES = ["C.scala"]
`,
			glob: `glob(INCLUDES, exclude = EXCLUDES)`,
			want: rule.GlobValue{Patterns: []string{"A.scala"}, Excludes: []string{"C.scala"}},
		},
		"complex value - not supported": {
			glob: `glob(get_include_list(), get_exclude_list())`,
			want: rule.GlobValue{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			content := fmt.Sprintf("test_rule(srcs = %s)", tc.glob)
			if tc.prelude != "" {
				content = tc.prelude + "\n\n" + content
			}
			file, err := rule.LoadData("<in-memory>", "BUILD", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			r := file.File.Rules("test_rule")[0]

			got := Parse(file, r.Attr("srcs").(*build.CallExpr))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Parse (-want +got):\n%s", diff)
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
					if err := os.WriteFile(abs, []byte(file.Content), os.ModePerm); err != nil {
						t.Fatal(err)
					}
				}
			}
			got := Apply(tc.glob, os.DirFS(tmpDir))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Apply (-want +got):\n%s", diff)
			}
		})
	}
}
