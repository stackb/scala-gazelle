package resolver

import (
	"flag"
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func init() {
	GlobalConflictResolverRegistry().PutConflictResolver(
		"scala_proto_grpc", &scalaProtoGrpcConflictResolver{},
	)
}

// scalaProtoGrpcConflictResolver implements a strategy where
type scalaProtoGrpcConflictResolver struct {
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *scalaProtoGrpcConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.ConflictResolver interface.
func (s *scalaProtoGrpcConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// ResolveConflict implements part of the resolver.ConflictResolver interface.
func (s *scalaProtoGrpcConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol) (*Symbol, bool) {
	if len(symbol.Conflicts) == 0 {
		panic(fmt.Sprintf("resolve conflict should not be called on non-conflicting symbols: %v", symbol))
	}

	// things I have in my head: if a user performs an import statement, we
	// really need to know the actual compiler scope to resolve it correctly.
	// which for the import 'proto._', the only true way is to know what symbols
	// from .proto were actually used, but we can't really know that from the
	// syntax tree.  Hence, the compiler.

	// criteria for this scenario to apply:
	// 1. conflicts labels must be of same repo/package
	//

	var isGrpc bool
	var isProto bool

	switch symbol.Type {
	case sppb.ImportType_PROTO_ENUM:
		fallthrough
	case sppb.ImportType_PROTO_ENUM_FIELD:
		fallthrough
	case sppb.ImportType_PROTO_MESSAGE:
		isProto = true
	case sppb.ImportType_PROTO_SERVICE:
		isGrpc = true
	}

	if !(isGrpc || isProto) {
		return nil, false
	}
	// if we find grpc, use that intead, proto dependency is not needed.
	// for _, sym := range symbol.Conflicts {

	// }

	return nil, false
}

func commonLabelNamePrefix(a, b label.Label) (string, bool) {
	if a.Repo != b.Repo {
		return "", false
	}
	if a.Pkg != b.Pkg {
		return "", false
	}

	r := a.Name
	s := b.Name
	if len(s) < len(r) {
		tmp := r
		r = s
		s = tmp
	}

	var index int
	for i := range r {
		if r[i] != s[i] {
			break
		}
		index++
	}

	return a.Name[0:index], true
}
