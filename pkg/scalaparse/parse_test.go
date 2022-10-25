package scalaparse

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

// See https://github.com/scalameta/scalameta/tree/main/semanticdb/integration/src/main/scala/example for syntax examples

func TestParse(t *testing.T) {
	for name, tc := range map[string]struct {
		label string
		files []testtools.FileSpec
		want  ExecResult
	}{
		"parse-error": {
			label: "//src/main/scala/app:app",
			files: []testtools.FileSpec{
				{
					Path:    "test/error.scala",
					Content: "packge foo",
				},
			},
			want: ExecResult{
				Label: "//src/main/scala/app:app",
				Srcs: []SourceFile{
					{
						Filename: "test/error.scala",
						Error:    "expected class or object definition identifier",
					},
				},
				Stderr: `{
  "error": "expected class or object definition identifier",
  "pos": {
    "start": 0,
    "end": 6
  },
  "lineNumber": 0,
  "columnNumber": 0
}
`,
			},
		},
		"simple": {
			label: "//src/main/scala/app:app",
			files: []testtools.FileSpec{
				{
					Path:    "test/A.scala",
					Content: "package a\n\nclass A{}",
				},
			},
			want: ExecResult{
				Label: "//src/main/scala/app:app",
				Srcs: []SourceFile{
					{
						Filename: "test/A.scala",
						Packages: []string{"a"},
						Classes:  []string{"a.A"},
						Names:    []string{"a"},
					},
				},
			},
		},
		"classes": {
			label: "//src/main/scala/app:app",
			files: []testtools.FileSpec{
				{
					Path: "test/classes.scala",
					Content: `
		package users
		class User {}
		`,
				},
			},
			want: ExecResult{
				Label: "//src/main/scala/app:app",
				Srcs: []SourceFile{
					{
						Filename: "test/classes.scala",
						Packages: []string{"users"},
						Classes:  []string{"users.User"},
						Names:    []string{"users"},
					},
				},
			},
		},
		"imports": {
			label: "//src/main/scala/app:app",
			files: []testtools.FileSpec{
				{
					Path: "test/imports.scala",
					Content: `
							import users._  // import everything from the users package
							import users.User  // import the class User
							import users.{User, UserPreferences}  // Only imports selected members
							import users.{UserPreferences => UPrefs}  // import and rename for convenience
							`,
				},
			},
			want: ExecResult{
				Label: "//src/main/scala/app:app",
				Srcs: []SourceFile{
					{
						Filename: "test/imports.scala",
						Imports:  []string{"users.User", "users.UserPreferences", "users._"},
						Names:    []string{"users"},
					},
				},
			},
		},

		"kitchen-sink": {
			label: "//src/main/scala/app:app",
			files: []testtools.FileSpec{
				{
					Path: "test/A.scala",
					Content: `
		package a

		import com.typesafe.config.ConfigFactory
		import com.typesafe.scalalogging.LazyLogging
		import scala.util.{Failure, Success}
		import java.time._
		import java.util.{Map}

		trait Trait {}
		abstract class AbstractClass extends Trait {}
		class ConcreteClass extends AbstractClass {}
		class AConfigFactory extends ConfigFactory {}
		object ALogger extends LazyLogging {
			type Id = String

		}
		case class CaseClass(
		    id: String,
		)
		`,
				},
			},
			want: ExecResult{
				Label: "//src/main/scala/app:app",
				Srcs: []SourceFile{
					{
						Filename: "test/A.scala",
						Packages: []string{"a"},
						Imports: []string{
							"com.typesafe.config.ConfigFactory",
							"com.typesafe.scalalogging.LazyLogging",
							"java.time._",
							"java.util.Map",
							"scala.util.Failure",
							"scala.util.Success",
						},
						Classes: []string{
							"a.AConfigFactory",
							"a.AbstractClass",
							"a.CaseClass",
							"a.ConcreteClass",
						},
						Objects: []string{"a.ALogger"},
						Traits:  []string{"a.Trait"},
						Names: []string{
							"ALogger",
							"a",
							"com",
							"config",
							"id",
							"java",
							"scala",
							"scalalogging",
							"time",
							"typesafe",
							"util",
						},
						Extends: map[string][]string{
							"class a.AConfigFactory": {"ConfigFactory"},
							"class a.AbstractClass":  {"Trait"},
							"class a.ConcreteClass":  {"AbstractClass"},
							"object a.ALogger":       {"LazyLogging"},
						},
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

			got, err := Parse(tc.label, filenames)
			if err != nil {
				t.Fatal("parse error:", got, err)
			}
			if got.ExitCode != 0 {
				t.Fatal("parse exit:", got, got.ExitCode)
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
