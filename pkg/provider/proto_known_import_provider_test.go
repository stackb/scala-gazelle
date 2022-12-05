package provider

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func TestProtoKnownImportProviderOnResolve(t *testing.T) {
	for name, tc := range map[string]struct {
		lang    string
		impLang string
		imports map[label.Label][]string
		want    []*resolver.KnownImport
	}{
		"degenerate": {},
		"hit": {
			lang:    "scala",
			impLang: "scala",
			imports: map[label.Label][]string{
				label.New("", "com/foo/bar/proto", "proto_scala_library"): {
					"com.foo.bar.proto.Message",
					"com.foo.bar.proto.Enum",
				},
			},
			want: []*resolver.KnownImport{
				{
					Type:   sppb.ImportType_CLASS,
					Import: "com.foo.bar.proto.Message",
					Label:  label.Label{Repo: "", Pkg: "com/foo/bar/proto", Name: "proto_scala_library"},
				},
				{
					Type:   sppb.ImportType_CLASS,
					Import: "com.foo.bar.proto.Enum",
					Label:  label.Label{Repo: "", Pkg: "com/foo/bar/proto", Name: "proto_scala_library"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			importProvider := &mockImportProvider{imports: tc.imports}
			importRegistry := &mockKnownImportRegistry{}

			p := NewStackbRulesProtoKnownImportProvider(
				tc.lang, tc.impLang,
				importProvider, importRegistry)
			p.OnResolve()

			if diff := cmp.Diff(tc.want, importRegistry.got); diff != "" {
				t.Errorf(".OnResolve (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtoKnownImportProviderCanProvide(t *testing.T) {
	for name, tc := range map[string]struct {
		lang      string
		imports   map[label.Label][]string
		from      label.Label
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
			importProvider := &mockImportProvider{imports: tc.imports}
			importRegistry := &mockKnownImportRegistry{}

			p := NewStackbRulesProtoKnownImportProvider(tc.lang, tc.lang, importProvider, importRegistry)
			p.OnResolve()

			got := p.CanProvide(tc.from, tc.indexFunc)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".CanProvide (-want +got):\n%s", diff)
			}
		})
	}
}
