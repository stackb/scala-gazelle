package resolver

import (
	"flag"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func init() {
	cr := &ProtoPackageWildcardImportConflictResolver{}
	GlobalConflictResolverRegistry().PutConflictResolver(cr.Name(), cr)
}

// ProtoPackageWildcardImportConflictResolver implements a strategy wherein a
// PROTO_PACKAGE symbol representing a wildcard import is "expanded" such that
// matching Names in the origin file are used to complete symbols in the given
// universe scope.
type ProtoPackageWildcardImportConflictResolver struct {
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *ProtoPackageWildcardImportConflictResolver) Name() string {
	return "proto_package_wildcard_import"
}

// RegisterFlags implements part of the resolver.ConflictResolver interface.
func (s *ProtoPackageWildcardImportConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags implements part of the resolver.ConflictResolver interface.
func (s *ProtoPackageWildcardImportConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// ResolveConflict implements part of the resolver.ConflictResolver interface.
// This implementation checks that the symbol is a proto package; if so, it uses
// the Names array of the source file to build a list of fully-qualified names
// that are found in the universe scope.  Matching symbol are added to the
// import map.
func (s *ProtoPackageWildcardImportConflictResolver) ResolveConflict(universe Universe, r *rule.Rule, imports ImportMap, imp *Import, symbol *Symbol, from label.Label) (*Symbol, bool) {
	if symbol.Type != sppb.ImportType_PROTO_PACKAGE {
		return nil, false
	}
	if imp.Source == nil {
		return nil, false
	}

	file := imp.Source
	prefix := strings.TrimSuffix(imp.Imp, "._")

	putImport := func(imp *Import) {
		if IsSelfImport(imp, from.Repo, from.Pkg, r.Name()) {
			return
		}
		if !imports.Has(imp.Imp) {
			imports.Put(imp)
		}
	}

	names := make([]string, 0)
	for _, name := range file.Names {
		fqn := prefix + "." + name
		if sym, ok := universe.GetSymbol(fqn); ok {
			isProtoSymbol := sym.Type == sppb.ImportType_PROTO_ENUM || sym.Type == sppb.ImportType_PROTO_MESSAGE || sym.Type == sppb.ImportType_PROTO_SERVICE
			if !isProtoSymbol {
				continue
			}
			putImport(NewResolvedNameImport(fqn, file, name, sym))
			names = append(names, name)
		}
	}

	if len(names) > 0 && wantSuggestions() {
		sort.Strings(names)
		log.Printf(
			"notice: in file %q the wildcard import %q should be replaced with:\nimport %s.{%s}",
			file.Filename,
			imp.Imp,
			prefix,
			strings.Join(names, ", "),
		)
	}

	return nil, len(names) > 0
}

func wantSuggestions() bool {
	if _, ok := os.LookupEnv("SCALA_GAZELLE_SUGGEST_PROTO_PACKAGE_WILDCARD_IMPORT_REPLACEMENTS"); ok {
		return true
	}
	return false
}
