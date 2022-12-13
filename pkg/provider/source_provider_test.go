package provider_test

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/mock"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	rmocks "github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	cmocks "github.com/stackb/scala-gazelle/pkg/scalacompile/mocks"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestScalaSourceProvider(t *testing.T) {
	for name, tc := range map[string]struct {
		args              []string
		kind              string
		from              label.Label
		testfiles         []string // name(s) of files under testdata/
		mockCompiledFiles []*sppb.File
		wantPutSymbols    []*resolver.Symbol
		wantFiles         []*sppb.File
	}{
		"GreeterClient.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/GreeterClient.scala"},
			mockCompiledFiles: []*sppb.File{
				{
					Symbols: []*sppb.Symbol{
						{Type: sppb.SymbolType_SYMBOL_OBJECT, Name: "foo"},
					},
				},
			},
			wantPutSymbols: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_OBJECT,
					Name:     "examples.helloworld.greeter.GreeterClient",
					Label:    label.Label{Name: "greeter_lib"},
					Provider: "scala_library",
				},
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "examples.helloworld.greeter",
					Label:    label.Label{Name: "greeter_lib"},
					Provider: "scala_library",
				},
			},
			wantFiles: []*sppb.File{
				{
					Filename: "testdata/GreeterClient.scala",
					Imports: []string{
						"akka.Done", "akka.NotUsed", "akka.actor.ActorSystem",
						"akka.grpc.GrpcClientSettings", "akka.stream.Materializer",
						"akka.stream.scaladsl.Source", "com.typesafe.scalalogging.LazyLogging",
						"examples.helloworld.greeter.proto.GreeterServiceClient",
						"examples.helloworld.greeter.proto.HelloReplyMessage",
						"examples.helloworld.greeter.proto.HelloRequest",
						"examples.helloworld.greeter.proto.MonitorHelloRequest",
						"scala.concurrent.Future", "scala.concurrent.duration._", "scala.util.Failure",
						"scala.util.Success",
					},
					Objects:  []string{"examples.helloworld.greeter.GreeterClient"},
					Packages: []string{"examples.helloworld.greeter"},
					Extends: map[string]*sppb.ClassList{
						"object examples.helloworld.greeter.GreeterClient": {Classes: []string{"com.typesafe.scalalogging.LazyLogging"}},
					},
					Symbols: []*sppb.Symbol{
						{Type: sppb.SymbolType_SYMBOL_OBJECT, Name: "foo"},
					},
				},
			},
		},
		"FullyQualified.scala": {
			kind:      "scala_library",
			from:      label.Label{Name: "greeter_lib"},
			testfiles: []string{"testdata/FullyQualified.scala"},
			mockCompiledFiles: []*sppb.File{
				{
					Symbols: []*sppb.Symbol{
						{Type: sppb.SymbolType_SYMBOL_OBJECT, Name: "foo"},
					},
				},
			},
			wantPutSymbols: []*resolver.Symbol{},
			wantFiles: []*sppb.File{
				{
					Filename: "testdata/FullyQualified.scala",
					Imports:  []string{"sk.ygor.stackoverflow.q53326545.macros.ExampleMacro.methodName"},
					Packages: []string{"foo"},
					Objects:  []string{"foo.Main"},
				},
			},
		},
		"UserId.scala": {
			kind:              "scala_library",
			from:              label.Label{Name: "greeter_lib"},
			testfiles:         []string{"testdata/UserId.scala"},
			mockCompiledFiles: []*sppb.File{},
			wantPutSymbols:    []*resolver.Symbol{},
			wantFiles: []*sppb.File{
				{
					Filename: "testdata/UserId.scala",
					Packages: []string{"common.types"},
					Classes:  []string{"common.types.UserId"},
					Objects:  []string{"common.types.UserId"},
					Names: []string{
						".value",
						"AnyVal",
						"Int",
						"UserId",
						"intTypeMapper",
						"scalapb.TypeMapper",
						"common.types",
						"value",
					},
					Extends: map[string]*sppb.ClassList{
						"class common.types.UserId": {Classes: []string{"AnyVal"}},
					},
				},
			},
		},
		"PostgresAccess.scala": {
			kind:              "scala_library",
			from:              label.Label{Name: "greeter_lib"},
			testfiles:         []string{"testdata/PostgresAccess.scala"},
			mockCompiledFiles: []*sppb.File{},
			wantPutSymbols:    []*resolver.Symbol{},
			wantFiles: []*sppb.File{
				{
					Filename: "testdata/UserId.scala",
					Packages: []string{"common.types"},
					Classes:  []string{"common.types.UserId"},
					Objects:  []string{"common.types.UserId"},
					Names: []string{
						".value",
						"AnyVal",
						"Int",
						"UserId",
						"intTypeMapper",
						"scalapb.TypeMapper",
						"common.types",
						"value",
					},
					Extends: map[string]*sppb.ClassList{
						"class common.types.UserId": {Classes: []string{"AnyVal"}},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			testutil.ListFiles(t, dir)

			known := rmocks.NewSymbolsCapturer(t)

			compiler := cmocks.NewCompiler(t)
			compiler.
				On("CompileScalaFiles", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.mockCompiledFiles, nil)

			p := provider.NewSourceProvider(compiler, func(msg string) {})

			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: dir,
			}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			if err := p.CheckFlags(fs, c, known.Registry); err != nil {
				t.Fatal(err)
			}

			files, err := p.ParseScalaFiles(tc.from, tc.kind, dir, tc.testfiles...)
			p.OnResolve()
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				if file.Error != "" {
					t.Fatal("parse rule file error:", file.Error)
				}
			}
			if diff := cmp.Diff(tc.wantPutSymbols, known.Got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantFiles, files,
				cmpopts.IgnoreUnexported(sppb.File{}, sppb.Symbol{}, sppb.ClassList{}),
			); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}

		})
	}
}
