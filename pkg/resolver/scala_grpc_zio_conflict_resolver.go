package resolver

import (
	"flag"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	grpcZioLabelSuffix = "_grpc_zio_scala_library"
)

func init() {
	cr := &ScalaGrpcZioConflictResolver{}
	GlobalConflictResolverRegistry().PutConflictResolver(cr.Name(), cr)
}

// ScalaGrpcZioConflictResolver implements a strategy
type ScalaGrpcZioConflictResolver struct {
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *ScalaGrpcZioConflictResolver) Name() string {
	return "scala_grpc_zio"
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *ScalaGrpcZioConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.ConflictResolver interface.
func (s *ScalaGrpcZioConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// ResolveConflict implements part of the resolver.ConflictResolver interface.
// This implementation deals with the specific scenario where a PROTO_PACKAGE is
// being provided by two rules: one is the "foo_proto_scala_library", the other
// is the "foo_grpc_scala_library".  The task is to determine if the rule is
// referencing *any* grpc-like symbols from the conflicting rule.  If they are
// using grpc, always resolve to conflict in favor of the grpc label, because
// that rule will include the protos anyway.  If they aren't using grpc, take
// the proto rule so that the rule does not take on additional unnecessary deps.
func (s *ScalaGrpcZioConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol) (*Symbol, bool) {
	if len(symbol.Conflicts) != 1 {
		return nil, false
	}

	a := symbol
	b := symbol.Conflicts[0]

	if a.Name != b.Name {
		return nil, false
	}

	// sort such that the grpc label is first, then assert that the other one
	// is the grpc_zio label.  If not, we cannot resolve this conflict.
	if !isGrpcLabel(a.Label.Name) {
		a, b = b, a
	}
	if !isGrpcZioLabel(b.Label.Name) {
		return nil, false
	}

	// search all imports for Zio symbols.  If we find one, assume we want Zio.
	// the string pattern we are looking for is a symbol name whose last dotted
	// name starts with "Zio" (e.g. 'com.contoso.proto.api.ZioUserService').
	var isUsingZio bool

	for _, imp := range imports.Values() {
		if imp.Symbol == nil || imp.Error != nil {
			continue
		}
		if imp.Symbol == symbol {
			continue
		}
		parts := strings.Split(imp.Symbol.Name, ".")
		last := parts[len(parts)-1]
		if !strings.HasPrefix(last, "Zio") {
			continue
		}
		isUsingZio = true
		break
	}

	if isUsingZio {
		return b, true
	} else {
		return a, true
	}
}

func isGrpcZioLabel(name string) bool {
	return strings.HasSuffix(name, grpcZioLabelSuffix)
}
