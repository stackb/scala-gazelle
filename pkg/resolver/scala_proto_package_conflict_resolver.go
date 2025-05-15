package resolver

import (
	"flag"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/rs/zerolog"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

const (
	protoLabelSuffix = "_proto_scala_library"
	grpcLabelSuffix  = "_grpc_scala_library"
)

// TODO(pcj): import these from 'github.com/stackb/rules_proto/pkg/rule/rules_scala' directly.
var serviceSuffixes = []string{
	"Grpc",
	"Client",
	"Handler",
	"Server",
	"PowerApi",
	"PowerApiHandler",
	"ClientPowerApi",
}

func init() {
	cr := &ScalaProtoPackageConflictResolver{}
	GlobalConflictResolverRegistry().PutConflictResolver(cr.Name(), cr)
}

// ScalaProtoPackageConflictResolver implements a strategy where
type ScalaProtoPackageConflictResolver struct {
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *ScalaProtoPackageConflictResolver) Name() string {
	return "scala_proto_package"
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *ScalaProtoPackageConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config, logger zerolog.Logger) {
}

// CheckFlags implements part of the resolver.ConflictResolver interface.
func (s *ScalaProtoPackageConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
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
func (s *ScalaProtoPackageConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol) (*Symbol, bool) {
	if len(symbol.Conflicts) != 1 {
		return nil, false
	}

	a := symbol
	b := symbol.Conflicts[0]

	if !(a.Type == sppb.ImportType_PROTO_PACKAGE && b.Type == sppb.ImportType_PROTO_PACKAGE) {
		return nil, false
	}
	if a.Name != b.Name {
		return nil, false
	}

	var isUsingGrpc bool
	pkg := a.Name

search:
	for _, imp := range imports.Values() {
		if imp.Symbol == nil || imp.Error != nil {
			continue
		}
		if imp.Symbol == symbol {
			continue
		}
		name := imp.Symbol.Name
		if !strings.HasPrefix(name, pkg) {
			continue
		}
		for _, suffix := range serviceSuffixes {
			if strings.HasSuffix(name, suffix) {
				isUsingGrpc = true
				break search
			}
		}
	}

	// sort such that the proto label is first, then assert that the other one
	// is the grpc label.  If not, we got confused.
	if !isProtoLabel(a.Label.Name) {
		a, b = b, a
	}
	if !isGrpcLabel(b.Label.Name) {
		return nil, false
	}

	if isUsingGrpc {
		return b, true
	} else {
		return a, true
	}
}

func isProtoLabel(name string) bool {
	return strings.HasSuffix(name, protoLabelSuffix)
}

func isGrpcLabel(name string) bool {
	return strings.HasSuffix(name, grpcLabelSuffix)
}
