package scalaparse

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestServerParse(t *testing.T) {
	for name, tc := range map[string]*struct {
		files []testtools.FileSpec
		in    sppb.ParseRequest
		want  sppb.ParseResponse
	}{
		"degenerate": {
			want: sppb.ParseResponse{
				Error: `bad request: expected '{ "filenames": [LIST OF FILES TO PARSE] }', but filenames list was not present`,
			},
		},
		"single file": {
			files: []testtools.FileSpec{
				{
					Path: "A.scala",
					Content: `package a
import java.util.HashMap

class Foo extends HashMap {
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "A.scala",
						Packages: []string{"a"},
						Classes:  []string{"a.Foo"},
						Imports:  []string{"java.util.HashMap"},
						Names:    []string{"a", "java", "util"},
						Extends: map[string]*sppb.ClassList{
							"class a.Foo": {
								Classes: []string{"java.util.HashMap"},
							},
						},
					},
				},
			},
		},
		"nested import": {
			files: []testtools.FileSpec{
				{
					Path: "Example.scala",
					Content: `
package example

import com.typesafe.scalalogging.LazyLogging
import corp.common.core.vm.utils.ArgProcessor

object Main extends LazyLogging {
	def main(args: Array[String]): Unit = {
	import corp.common.core.reports.DotFormatReport
	ArgProcessor.process(args)
	logger.info(DotFormatReport(new BlendTestService).dotForm())
	}
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Example.scala",
						Packages: []string{"example"},
						Objects:  []string{"example.Main"},
						Imports: []string{
							"com.typesafe.scalalogging.LazyLogging",
							"corp.common.core.reports.DotFormatReport",
							"corp.common.core.vm.utils.ArgProcessor",
						},
						Extends: map[string]*sppb.ClassList{
							"object example.Main": {
								Classes: []string{"com.typesafe.scalalogging.LazyLogging"},
							},
						},
					},
				},
			},
		},
		"extends with": {
			files: []testtools.FileSpec{
				{
					Path: "FooTest.scala",
					Content: `
package foo.test

import org.scalatest.{FlatSpec, Matchers}
import java.time.{LocalDate, LocalTime}

class FooTest extends FlatSpec with Matchers {
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "FooTest.scala",
						Packages: []string{"foo.test"},
						Classes:  []string{"foo.test.FooTest"},
						Imports: []string{
							"java.time.LocalDate",
							"java.time.LocalTime",
							"org.scalatest.FlatSpec",
							"org.scalatest.Matchers",
						},
						Extends: map[string]*sppb.ClassList{
							"class foo.test.FooTest": {
								Classes: []string{
									"org.scalatest.FlatSpec",
									"org.scalatest.Matchers",
								},
							},
						},
					},
				},
			},
		},
		"nested import rename": {
			files: []testtools.FileSpec{
				{
					Path: "Palette.scala",
					Content: `
package color

import java.awt.Color

object Palette {
  val random100: MandelPalette = {
    import scala.util.Random.{nextInt => rint}
    Palette(100, Seq.tabulate[Color](100)(_ => new Color(rint(255), rint(255), rint(255))).toArray)
  }
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Palette.scala",
						Packages: []string{"color"},
						Objects:  []string{"color.Palette"},
						Imports: []string{
							"java.awt.Color",
							"scala.util.Random.nextInt",
						},
					},
				},
			},
		},
		"nested import same package": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
package example

import akka.actor.ActorSystem

object MainContext {
	implicit var asys: ActorSystem = _
}
  
object Main {
	private def makeRequest(params: Map[String, String]): Unit = {
		import MainContext._
	}	
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Packages: []string{"example"},
						Objects: []string{
							"example.Main",
							"example.MainContext",
						},
						Imports: []string{
							"akka.actor.ActorSystem",
						},
					},
				},
			},
		},
	} {
		// if name != "nested import" {
		// 	continue
		// }
		t.Run(name, func(t *testing.T) {
			tmpDir, err := bazel.NewTmpDir("")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			files := mustWriteTestFiles(t, tmpDir, tc.files)
			tc.in.Filenames = files

			server := NewScalametaParser()
			if err := server.Start(); err != nil {
				t.Fatal("server start:", err)
			}
			defer server.Stop()

			got, err := server.Parse(context.Background(), &tc.in)
			if err != nil {
				t.Fatal(err)
			}
			got.ElapsedMillis = 0

			// remove tmpdir prefix and zero the time delta for diff comparison
			for i := range got.Files {
				if strings.HasPrefix(got.Files[i].Filename, tmpDir) {
					got.Files[i].Filename = got.Files[i].Filename[len(tmpDir)+1:]
				}
			}

			if diff := cmp.Diff(&tc.want, got, cmpopts.IgnoreUnexported(
				sppb.ParseResponse{},
				sppb.File{},
				sppb.ClassList{},
			), cmpopts.IgnoreFields(sppb.File{}, "Names")); diff != "" {
				t.Errorf(".Parse (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetFreePort(t *testing.T) {
	got, err := getFreePort()
	if err != nil {
		t.Fatal(err)
	}
	if got == 0 {
		t.Error("expected non-zero port number")
	}
}

func TestNewHttpScalaParseRequest(t *testing.T) {
	for name, tc := range map[string]struct {
		url      string
		in       *sppb.ParseRequest
		want     *http.Request
		wantBody string
	}{
		"prototypical": {
			url: "http://localhost:3000",
			in: &sppb.ParseRequest{
				Filenames: []string{"A.scala", "B.scala"},
			},
			want: &http.Request{
				Method:        "POST",
				URL:           mustParseURL(t, "http://localhost:3000"),
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Header:        http.Header{"Content-Type": {"application/json"}},
				ContentLength: 36, // or 35, see below!
				Host:          "localhost:3000",
			},
			wantBody: `{"filenames":["A.scala","B.scala"]}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := newHttpParseRequest(tc.url, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			body, err := ioutil.ReadAll(got.Body)
			if err != nil {
				t.Fatal(err)
			}
			// remove all whitespace (and ignore content length) for the test:
			// seeing CI failures between macos (M1) and linux.  Very strange!
			gotBody := strings.ReplaceAll(string(body), " ", "")
			if diff := cmp.Diff(tc.want, got,
				cmpopts.IgnoreUnexported(http.Request{}),
				cmpopts.IgnoreFields(http.Request{}, "GetBody", "Body", "ContentLength"),
			); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantBody, gotBody); diff != "" {
				t.Errorf("body (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewHttpParseRequestError(t *testing.T) {
	for name, tc := range map[string]struct {
		url  string
		in   *sppb.ParseRequest
		want error
	}{
		"missing-url": {
			want: fmt.Errorf("rpc error: code = InvalidArgument desc = request URL is required"),
		},
		"missing-request": {
			url:  "http://localhost:3000",
			want: fmt.Errorf("rpc error: code = InvalidArgument desc = ParseRequest is required"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, got := newHttpParseRequest(tc.url, tc.in)
			if got == nil {
				t.Fatalf("error was expected: %v", tc.want)
			}
			if diff := cmp.Diff(tc.want.Error(), got.Error()); diff != "" {
				t.Errorf("newHttpScalaParseRequest error (-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url parse error: %v", err)
	}
	return u
}

func mustWriteTestFiles(t *testing.T, tmpDir string, files []testtools.FileSpec) []string {
	var filenames []string
	for _, file := range files {
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
	return filenames
}
