package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestScalaGrpcZioConflictResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		symbol  resolver.Symbol
		rule    rule.Rule
		imports resolver.ImportMap
		imp     resolver.Import
		want    *resolver.Symbol
		wantOk  bool
	}{
		"degenerate": {
			symbol: resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
			},
		},
		"selects-symbol-from-grpc-label-when-no-zio-imports": {
			symbol: resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api.UserGrpc",
						Type:  sppb.ImportType_PROTO_SERVICE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
					},
				},
			},
			want: &resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api.UserGrpc",
						Type:  sppb.ImportType_PROTO_SERVICE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
					},
				},
			},
			wantOk: true,
		},
		"selects-symbol-from-grpc-label-when-no-zio-imports-unsorted": {
			symbol: resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api.UserGrpc",
						Type:  sppb.ImportType_PROTO_SERVICE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
					},
				},
			},
			want: &resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
			},
			wantOk: true,
		},
		"selects-symbol-from-grpc-zio-label-when-zio-import-present-suffix-grpc": {
			symbol: resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api.UserGrpc",
						Type:  sppb.ImportType_PROTO_SERVICE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
					},
				},
			},
			imports: resolver.NewImportMap(
				&resolver.Import{
					Symbol: &resolver.Symbol{Name: "proto.api.ZioUser"},
				},
			),
			wantOk: true,
			want: &resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
			},
		},
		"selects-symbol-from-grpc-zio-label-when-zio-import-present-contains-grpc": {
			symbol: resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api.UserGrpc",
						Type:  sppb.ImportType_PROTO_SERVICE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
					},
				},
			},
			imports: resolver.NewImportMap(
				&resolver.Import{
					// in this case, the string .Zio is not the suffix of the
					// last symbol, but somewhere higher up in the string.
					Symbol: &resolver.Symbol{Name: "proto.api.ZioUser.X"},
				},
			),
			wantOk: true,
			want: &resolver.Symbol{
				Name:  "proto.api.UserGrpc",
				Type:  sppb.ImportType_PROTO_SERVICE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_zio_scala_library"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := mocks.NewUniverse(t)
			rslv := resolver.ScalaGrpcZioConflictResolver{}
			imports := tc.imports
			if imports == nil {
				imports = resolver.NewImportMap()
			}
			got, gotOk := rslv.ResolveConflict(universe, &tc.rule, imports, &tc.imp, &tc.symbol)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ok (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
