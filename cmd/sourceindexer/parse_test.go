package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	for name, tc := range map[string]struct {
		label string
		files []testtools.FileSpec
		want  ParseResult
	}{
		// "degenerate": {},
		"simple": {
			label: "//src/main/scala/app:app",
			files: []testtools.FileSpec{
				{Path: "test/A.scala", Content: "package a\n\nclass A{}"},
			},
			want: ParseResult{
				Label: "//src/main/scala/app:app",
				Srcs: []SourceFile{
					{
						Filename: "test/A.scala",
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, err := bazel.NewTmpDir("")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			var filenames []string
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
				filenames = append(filenames, abs)
			}

			got, exitCode, err := Parse(tc.label, filenames)
			if err != nil {
				t.Fatal("parse error:", got, exitCode, err)
			}
			if exitCode != 0 {
				t.Fatal("parse exit:", got, exitCode)
			}
			// strip abs filename result for purposes of test comparison
			for i := range got.Srcs {
				if strings.HasPrefix(got.Srcs[i].Filename, tmpDir) {
					got.Srcs[i].Filename = got.Srcs[i].Filename[len(tmpDir)+1:]
				}
			}

			transformJSON := cmp.FilterValues(func(x, y []byte) bool {
				return json.Valid(x) && json.Valid(y)
			}, cmp.Transformer("ParseJSON", func(in []byte) (out interface{}) {
				if err := json.Unmarshal(in, &out); err != nil {
					panic(err) // should never occur given previous filter to ensure valid JSON
				}
				return out
			}))

			if diff := cmp.Diff(&tc.want, got, transformJSON); diff != "" {
				t.Errorf("Parse out (-want +got):\n%s", diff)
			}
		})
	}
}
