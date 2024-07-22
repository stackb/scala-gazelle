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

func TestScalaProtoPackageConflictResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		symbol  resolver.Symbol
		rule    rule.Rule
		imports resolver.ImportMap
		imp     resolver.Import
		from    label.Label
		want    *resolver.Symbol
		wantOk  bool
	}{
		"degenerate": {
			symbol: resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_proto_scala_library"},
			},
		},
		"selects-symbol-from-proto-label-when-no-grpc-imports": {
			symbol: resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_proto_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api",
						Type:  sppb.ImportType_PROTO_PACKAGE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
					},
				},
			},
			wantOk: true,
			want: &resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_proto_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api",
						Type:  sppb.ImportType_PROTO_PACKAGE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
					},
				},
			},
		},
		"selects-symbol-from-proto-label-unsorted": {
			symbol: resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api",
						Type:  sppb.ImportType_PROTO_PACKAGE,
						Label: label.Label{Pkg: "proto/api", Name: "user_proto_scala_library"},
					},
				},
			},
			wantOk: true,
			want: &resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_proto_scala_library"},
			},
		},
		"selects-symbol-from-grpc-label-when-grpc-import-present-suffix-grpc": {
			symbol: resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_proto_scala_library"},
				Conflicts: []*resolver.Symbol{
					{
						Name:  "proto.api",
						Type:  sppb.ImportType_PROTO_PACKAGE,
						Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
					},
				},
			},
			imports: resolver.NewImportMap(
				&resolver.Import{
					Symbol: &resolver.Symbol{Name: "proto.api.UserGrpc"},
				},
			),
			wantOk: true,
			want: &resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := mocks.NewUniverse(t)
			resolver := resolver.ScalaProtoPackageConflictResolver{}
			got, gotOk := resolver.ResolveConflict(universe, &tc.rule, tc.imports, &tc.imp, &tc.symbol, tc.from)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ok (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
