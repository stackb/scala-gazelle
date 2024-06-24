package resolver_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestProtoPackageWildcardImportConflictResolver(t *testing.T) {
	for name, tc := range map[string]struct {
		universe    resolver.Universe
		symbol      resolver.Symbol
		rule        rule.Rule
		imports     resolver.ImportMap
		imp         resolver.Import
		from        label.Label
		want        *resolver.Symbol
		wantImports resolver.ImportMap
		wantOk      bool
	}{
		"degenerate": {
			universe: mocks.NewUniverse(t),
			symbol: resolver.Symbol{
				Name:  "proto.api",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
			},
		},
		"typical": {
			universe: func() resolver.Universe {
				u := mocks.NewUniverse(t)
				u.On("GetSymbol", "proto.api.User").Return(
					&resolver.Symbol{
						Name: "proto.api.User",
					},
					true,
				)
				u.On("GetSymbol", "proto.api.Group").Return(
					&resolver.Symbol{
						Name: "proto.api.Group",
					},
					true,
				)
				return u
			}(),
			from: label.New("", "api", "scala"),
			symbol: resolver.Symbol{
				Name:  "proto.api._",
				Type:  sppb.ImportType_PROTO_PACKAGE,
				Label: label.Label{Pkg: "proto/api", Name: "user_grpc_scala_library"},
			},
			imp: resolver.Import{
				Imp: "proto.api._",
				Source: &sppb.File{
					Names: []string{"User", "Group"},
				},
			},
			imports: resolver.NewImportMap(),
			wantImports: resolver.NewImportMap(
				&resolver.Import{
					Imp:  "proto.api.User",
					Kind: sppb.ImportKind_RESOLVED_NAME,
					Source: &sppb.File{
						Names: []string{"User", "Group"},
					},
					Src: "User",
					Symbol: &resolver.Symbol{
						Name: "proto.api.User",
					},
				},
				&resolver.Import{
					Imp:  "proto.api.Group",
					Kind: sppb.ImportKind_RESOLVED_NAME,
					Source: &sppb.File{
						Names: []string{"User", "Group"},
					},
					Src: "Group",
					Symbol: &resolver.Symbol{
						Name: "proto.api.Group",
					},
				},
			),
			wantOk: true,
			want:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			rslv := resolver.ProtoPackageWildcardImportConflictResolver{}
			got, gotOk := rslv.ResolveConflict(tc.universe, &tc.rule, tc.imports, &tc.imp, &tc.symbol, tc.from)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("got ok (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantImports, tc.imports, cmpopts.IgnoreUnexported(sppb.File{}, resolver.Import{})); diff != "" {
				t.Errorf("got importmap (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("got symbol (-want +got):\n%s", diff)
			}
		})
	}
}
