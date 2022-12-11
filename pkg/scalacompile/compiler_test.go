package scalacompile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// TestScalaCompileResponse tests translation of an XML response from the
// compiler to a CompileSpec.
func TestScalaCompileResponse(t *testing.T) {
	for name, tc := range map[string]struct {
		dir          string
		filename     string
		mockResponse string
		want         *ScalaCompileSpec
	}{
		"ok": {
			filename: "lib/App.scala",
			mockResponse: `
<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<compileResponse>
  <diagnostic line="57" sev="ERROR" source="lib/App.scala">not found: type Greeter</diagnostic>
  <diagnostic line="67" sev="ERROR" source="lib/App.scala">not found: type Greeter</diagnostic>
</compileResponse>
`,
			want: &ScalaCompileSpec{
				NotFound: []*NotFoundSymbol{{Name: "Greeter", Kind: "type"}},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				res.WriteHeader(http.StatusOK)
				res.Write([]byte(tc.mockResponse))
			}))
			defer testServer.Close()

			compiler := &Compiler{
				backendRawURL: testServer.URL,
			}

			if err := compiler.initHTTPClient(); err != nil {
				t.Fatal(err)
			}

			got, err := compiler.Compile(tc.dir, tc.filename)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("completions (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseScalaFileSpec checks that we can correctly read the JSON.
func TestParseScalaFileSpec(t *testing.T) {
	for name, tc := range map[string]struct {
		dir          string
		filename     string
		mockResponse string
		want         sppb.File
	}{
		"ok": {
			filename: "com/foo/Utils.scala",
			mockResponse: `
{
	"filename": "com/foo/Utils.scala",
	"packages": [
		"com.foo"
	],
	"imports": [
		"com.typesafe.scalalogging.LazyLogging"
	],
	"traits": [
		"com.foo.RationalUtils"
	],
	"objects": [
		"com.foo.RationalUtils"
	],
	"names": [
		"BigDecimal"
	],
	"extends": {
		"object com.foo.RationalUtils": {
			"classes": ["RationalPriceUtils"]
		},
		"object com.foo.RationalPriceUtils": {
			"classes": ["RationalPriceUtils"]
		},
		"trait com.foo.RationalPriceUtils": {
			"classes": ["RationalUtils"]
		},
		"trait com.foo.RationalUtils": {
			"classes": ["LazyLogging"]
		}
	}
}
`,
			want: sppb.File{
				Filename: "com/foo/Utils.scala",
				Imports:  []string{"com.typesafe.scalalogging.LazyLogging"},
				Packages: []string{"com.foo"},
				Objects:  []string{"com.foo.RationalUtils"},
				Traits:   []string{"com.foo.RationalUtils"},
				Names:    []string{"BigDecimal"},
				Extends: map[string]*sppb.ClassList{
					"object com.foo.RationalPriceUtils": {Classes: []string{"RationalPriceUtils"}},
					"object com.foo.RationalUtils":      {Classes: []string{"RationalPriceUtils"}},
					"trait com.foo.RationalPriceUtils":  {Classes: []string{"RationalUtils"}},
					"trait com.foo.RationalUtils":       {Classes: []string{"LazyLogging"}},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var got sppb.File
			if err := json.Unmarshal([]byte(tc.mockResponse), &got); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreUnexported(
				sppb.File{},
				sppb.ClassList{},
			)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

// TestScalaCompileResponse tests translation of an XML response from the
// compiler to a CompileSpec.
func TestScalaCompile(t *testing.T) {
	for name, tc := range map[string]struct {
		files    []testtools.FileSpec
		dir      string
		filename string
		request  CompileRequest
		want     CompileResponse
	}{
		"degenerate": {
			dir:      "",
			filename: "lib/App.scala",
		},
	} {
		t.Run(name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				res.WriteHeader(http.StatusOK)
				res.Write([]byte(tc.mockResponse))
			}))
			defer testServer.Close()

			compiler := &Compiler{
				backendRawURL: testServer.URL,
			}

			if err := compiler.initHTTPClient(); err != nil {
				t.Fatal(err)
			}

			got, err := compiler.Compile(tc.dir, tc.filename)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("completions (-want +got):\n%s", diff)
			}
		})
	}
}
