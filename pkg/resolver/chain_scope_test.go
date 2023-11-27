package resolver

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestChainScope(t *testing.T) {
	for name, tc := range map[string]struct {
		scopes []Scope
		name   string
		want   *Symbol
	}{
		"degenerate": {},
		"miss": {
			name: "examples.helloworld.greeter.GreeterServiceImpl",
			scopes: func() []Scope {
				root := NewTrieScope("test")
				root.PutSymbol(
					makeSymbol(sppb.ImportType_PROTO_PACKAGE, "examples.helloworld.greeter.proto", label.Label{Pkg: "examples/helloworld/greeter/proto", Name: "examples_helloworld_greeter_proto_grpc_scala_library"}),
				)
				scope, _ := root.GetScope("examples.helloworld.greeter.proto")
				return []Scope{scope}
			}(),
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scope := NewChainScope(tc.scopes...)
			got, _ := scope.GetSymbol(tc.name)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
