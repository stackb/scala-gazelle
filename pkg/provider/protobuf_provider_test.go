package provider_test

import (
	"flag"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/provider"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
)

func TestProtoSymbolProviderOnResolve(t *testing.T) {

	for name, tc := range map[string]struct {
		imports map[string]map[label.Label][]string
		want    []*resolver.Symbol
	}{
		"degenerate": {},
		"hit": {
			imports: map[string]map[label.Label][]string{
				"message": {
					label.New("", "com/foo/bar/proto", "proto_scala_library"): {
						"com.foo.bar.proto.Message",
					},
				},
				"enum": {
					label.New("", "com/foo/bar/proto", "proto_scala_library"): {
						"com.foo.bar.proto.Enum",
					},
				},
			},
			want: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_PROTO_ENUM,
					Name:     "com.foo.bar.proto.Enum",
					Label:    label.Label{Repo: "", Pkg: "com/foo/bar/proto", Name: "proto_scala_library"},
					Provider: "protobuf",
				},
				{
					Type:     sppb.ImportType_PROTO_MESSAGE,
					Name:     "com.foo.bar.proto.Message",
					Label:    label.Label{Repo: "", Pkg: "com/foo/bar/proto", Name: "proto_scala_library"},
					Provider: "protobuf",
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			known := mocks.NewSymbolsCapturer(t)

			p := provider.NewProtobufProvider(scalaName, scalaName, func(lang, impLang string) map[label.Label][]string {
				return tc.imports[impLang]
			})

			c := config.New()
			flags := flag.NewFlagSet(scalaName, flag.ExitOnError)
			p.CheckFlags(flags, c, known.Registry)

			p.OnResolve()

			if diff := cmp.Diff(tc.want, known.Got); diff != "" {
				t.Errorf(".OnResolve (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtoSymbolProviderCanProvide(t *testing.T) {
	for name, tc := range map[string]struct {
		lang      string
		imports   map[string]map[label.Label][]string
		from      label.Label
		expr      build.Expr
		indexFunc func(from label.Label) (*rule.Rule, bool)
		want      bool
	}{
		"degenerate case": {},
		"managed proto label": {
			lang: "scala",
			from: label.New("", "example", "foo_proto_scala_library"),
			want: true,
		},
		"managed grpc label": {
			lang: "scala",
			from: label.New("", "example", "foo_grpc_scala_library"),
			want: true,
		},
		"unmanaged non-proto/non-grpc label": {
			lang: "scala",
			from: label.New("", "example", "foo_scala_library"),
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := mocks.NewScope(t)

			p := provider.NewProtobufProvider(tc.lang, tc.lang, func(lang, impLang string) map[label.Label][]string {
				return tc.imports[lang]
			})
			c := config.New()
			flags := flag.NewFlagSet(scalaName, flag.ExitOnError)
			p.CheckFlags(flags, c, scope)
			p.OnResolve()

			got := p.CanProvide(&resolver.ImportLabel{Label: tc.from}, tc.expr, tc.indexFunc, tc.from)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".CanProvide (-want +got):\n%s", diff)
			}
		})
	}
}
