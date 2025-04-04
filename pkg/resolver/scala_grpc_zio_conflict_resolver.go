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
// The resolver attempts to decide if the grpc-zio label should be chosen in
// preference to the non-zio grpc label.
func (s *ScalaGrpcZioConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol) (*Symbol, bool) {
	// make list of all symbols
	symbols := append(symbol.Conflicts, symbol)

	// ensure we have both types needed for comparison
	grpcSymbol := firstSymbol(symbols, isGrpcLabel)
	if grpcSymbol == nil {
		return nil, false
	}
	grpcZioSymbol := firstSymbol(symbols, isGrpcZioLabel)
	if grpcZioSymbol == nil {
		return nil, false
	}

	// must be conflicted over same name
	if grpcSymbol.Name != grpcZioSymbol.Name {
		return nil, false
	}

	// search all imports for Zio symbols.  If we find one, assume we want Zio.
	// the string pattern we are looking for is a symbol name whose last dotted
	// name starts with "Zio" (e.g. 'com.contoso.proto.api.ZioUserService').
	for _, imp := range imports.Values() {
		if imp.Symbol == nil || imp.Error != nil {
			continue
		}
		if imp.Symbol == symbol {
			continue
		}
		if !strings.Contains(imp.Symbol.Name, ".Zio") {
			continue
		}
		return grpcZioSymbol, true
	}

	// did not find a zio like activity in the imports, so use non-zio one.
	return grpcSymbol, true
}

func firstSymbol(symbols []*Symbol, labelNamePredicate func(string) bool) *Symbol {
	for _, sym := range symbols {
		if labelNamePredicate(sym.Label.Name) {
			return sym
		}
	}
	return nil
}

func isGrpcZioLabel(name string) bool {
	return strings.HasSuffix(name, grpcZioLabelSuffix)
}
