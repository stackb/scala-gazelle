package scalacompile

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	rmocks "github.com/stackb/scala-gazelle/pkg/resolver/mocks"
)

// // TestScalaCompileResponse tests translation of a grpc response from the
// // compiler to a CompileSpec.
// func TestScalaCompileResponse(t *testing.T) {
// 	for name, tc := range map[string]struct {
// 		dir          string
// 		filename     string
// 		mockResponse string
// 		want         []*sppb.File
// 	}{
// 		"ok": {
// 			filename: "lib/App.scala",
// 			mockResponse: `
// <?xml version="1.0" encoding="UTF-8" standalone="no"?>
// <compileResponse>
//   <diagnostic line="57" sev="ERROR" source="lib/App.scala">not found: type Greeter</diagnostic>
//   <diagnostic line="67" sev="ERROR" source="lib/App.scala">not found: type Greeter</diagnostic>
// </compileResponse>
// `,
// 		},
// 	} {
// 		t.Run(name, func(t *testing.T) {
// 			testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
// 				res.WriteHeader(http.StatusOK)
// 				res.Write([]byte(tc.mockResponse))
// 			}))
// 			defer testServer.Close()
// 			compiler := &ScalacCompilerService{
// 				backendUrl: testServer.URL,
// 			}

// 			if err := compiler.start(); err != nil {
// 				t.Fatal(err)
// 			}

// 			got, err := compiler.CompileScala(label.NoLabel, "scala_library", tc.dir, tc.filename)
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			if diff := cmp.Diff(tc.want, got); diff != "" {
// 				t.Errorf("completions (-want +got):\n%s", diff)
// 			}
// 		})
// 	}
// }

func TestScalacCompiler(t *testing.T) {
	for name, tc := range map[string]struct {
		kind      string
		from      label.Label
		testfiles []string // name(s) of files under testdata/
		want      []*sppb.File
	}{
		"GreeterClient.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/GreeterClient.scala"},
			want: []*sppb.File{
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
		"FullyQualified.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/FullyQualified.scala"},
			want: []*sppb.File{
				{
					Filename: "testdata/FullyQualified.scala",
					Symbols: []*sppb.Symbol{
						{Type: sppb.SymbolType_SYMBOL_PACKAGE, Name: "java?dx"},
						{Type: sppb.SymbolType_SYMBOL_VALUE, Name: "sk"},
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
				"-scalac_jar_path=./scalacserver.jar",
				"-scalac_java_bin_path=../../external/local_jdk/bin/java",
				"-scalac_backend_port=8040",
				"-scalac_backend_dial_timeout=10s",
			}); err != nil {
				t.Fatal(err)
			}

			if err := compiler.CheckFlags(flags, c, rmocks.NewScope(t)); err != nil {
				t.Fatal(err)
			}

			// if err := compiler.Start(); err != nil {
			// 	t.Fatal(err)
			// }

			got, err := compiler.CompileScalaFiles(tc.from, dir, tc.testfiles...)
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