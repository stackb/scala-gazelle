package scalacompile

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// TestScalaCompileResponse tests translation of an XML response from the
// compiler to a CompileSpec.
func SkipTestScalaCompileResponse(t *testing.T) {
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
			compiler := &ScalacCompilerService{
				backendUrl: testServer.URL,
			}

			if err := compiler.start(); err != nil {
				t.Fatal(err)
			}

			got, err := compiler.CompileScala(label.NoLabel, "scala_library", tc.dir, tc.filename)
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

func TestCompiler(t *testing.T) {
	for name, tc := range map[string]struct {
		kind      string
		from      label.Label
		testfiles []string // name(s) of files under testdata/
		want      *sppb.Rule
	}{
		"GreeterClient.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/GreeterClient.scala"},
			want: &sppb.Rule{
				Label: "//:greeter_lib",
				Kind:  "scala_library",
				Files: []*sppb.File{
					{
						Filename: "testdata/GreeterClient.scala",
						Symbols: []*sppb.Symbol{
							{Type: sppb.SymbolType_SYMBOL_OBJECT, Name: "akka"},
							{Type: sppb.SymbolType_SYMBOL_PACKAGE, Name: "com?typesafe"},
							{Type: sppb.SymbolType_SYMBOL_PACKAGE, Name: "examples.helloworld.greeter?proto"},
							{Type: sppb.SymbolType_SYMBOL_TYPE, Name: "LazyLogging"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "ActorSystem"},
							{Type: sppb.SymbolType_SYMBOL_TYPE, Name: "Materializer"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "Materializer"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "GreeterServiceClient"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "GrpcClientSettings"},
							{Type: sppb.SymbolType_SYMBOL_TYPE, Name: "GreeterServiceClient"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "logger"},
							{Type: sppb.SymbolType_SYMBOL_TYPE, Name: "Source"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "Source"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "HelloRequest"},
							{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "NotUsed"},
							{Type: sppb.SymbolType_SYMBOL_TYPE, Name: "Done"},
						},
					},
				},
			},
		},
		"FullyQualified.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/FullyQualified.scala"},
			want: &sppb.Rule{
				Label: "//:greeter_lib",
				Kind:  "scala_library",
				Files: []*sppb.File{
					{
						Filename: "testdata/FullyQualified.scala",
						Symbols:  []*sppb.Symbol{},
					},
				},
			},
		},
	} {
		if name != "FullyQualified.scala" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			// testutil.ListFiles(t, "../..")

			compiler := NewScalacCompiler()

			flags := flag.NewFlagSet("scala", flag.ExitOnError)
			c := &config.Config{
				WorkDir: dir,
			}

			compiler.RegisterFlags(flags, "update", c)
			if err := flags.Parse([]string{
				"-scala_compiler_jar_path=./scalacserver.jar",
				"-scala_compiler_java_bin_path=../../external/local_jdk/bin/java",
				"-scala_compiler_backend_port=8040",
				"-scala_compiler_backend_dial_timeout=10s",
			}); err != nil {
				t.Fatal(err)
			}

			if err := compiler.CheckFlags(flags, c); err != nil {
				t.Fatal(err)
			}

			// if err := compiler.Start(); err != nil {
			// 	t.Fatal(err)
			// }

			got, err := compiler.CompileScala(tc.from, tc.kind, dir, tc.testfiles...)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("got: %+v", got)
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreUnexported(
				sppb.Rule{},
				sppb.File{},
				sppb.Symbol{},
			)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

// // TestScalaCompileResponse tests translation of an XML response from the
// // compiler to a CompileSpec.
// func TestScalaCompile(t *testing.T) {
// 	for name, tc := range map[string]struct {
// 		files    []testtools.FileSpec
// 		dir      string
// 		filename string
// 		request  CompileRequest
// 		want     CompileResponse
// 	}{
// 		"degenerate": {
// 			dir:      "",
// 			filename: "lib/App.scala",
// 		},
// 	} {
// 		t.Run(name, func(t *testing.T) {
// 			testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
// 				res.WriteHeader(http.StatusOK)
// 				res.Write([]byte(tc.mockResponse))
// 			}))
// 			defer testServer.Close()

// 			compiler := &Compiler{
// 				backendRawURL: testServer.URL,
// 			}

// 			if err := compiler.initHTTPClient(); err != nil {
// 				t.Fatal(err)
// 			}

// 			got, err := compiler.Compile(tc.dir, tc.filename)
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			if diff := cmp.Diff(tc.want, got); diff != "" {
// 				t.Errorf("completions (-want +got):\n%s", diff)
// 			}
// 		})
// 	}
// }
