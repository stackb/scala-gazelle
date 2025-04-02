package scala

import (
	"os"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
)

func TestScalaRuleExports(t *testing.T) {
	makeImportSpec := func(imp string) resolve.ImportSpec {
		return resolve.ImportSpec{Lang: scalaLangName, Imp: imp}
	}

	for name, tc := range map[string]struct {
		rule  *rule.Rule
		from  label.Label
		files []*sppb.File
		want  []resolve.ImportSpec
	}{
		"degenerate": {
			rule: rule.NewRule("scala_library", "somelib"), // rule must not be nil
			want: []resolve.ImportSpec{},
		},
		"known exports": {
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Imports:  []string{"com.foo.Bar"},
					Packages: []string{"com.foo"},
					Classes:  []string{"com.foo.ClassA", "com.foo.ClassB"},
					Objects:  []string{"com.foo.ObjectA", "com.foo.ObjectB"},
					Traits:   []string{"com.foo.TraitA", "com.foo.TraitB"},
					Types:    []string{"com.foo.TypeA", "com.foo.TypeB"},
					Vals:     []string{"com.foo.ValA", "com.foo.ValB"},
				},
			},
			want: []resolve.ImportSpec{
				makeImportSpec("com.foo.ClassA"),
				makeImportSpec("com.foo.ClassB"),
				makeImportSpec("com.foo.ObjectA"),
				makeImportSpec("com.foo.ObjectB"),
				makeImportSpec("com.foo.TraitA"),
				makeImportSpec("com.foo.TraitB"),
				makeImportSpec("com.foo.TypeA"),
				makeImportSpec("com.foo.TypeB"),
				makeImportSpec("com.foo.ValA"),
				makeImportSpec("com.foo.ValB"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := mocks.NewUniverse(t)
			scope := mocks.NewScope(t)

			scope.
				On("PutSymbol", mock.Anything).
				Maybe().
				Return(nil)

			c := config.New()
			sc := scalaconfig.New(zerolog.New(os.Stderr), universe, c, "")

			ctx := &scalaRuleContext{
				rule:        tc.rule,
				scalaConfig: sc,
				resolver:    universe,
				scope:       universe,
			}

			scalaRule := newScalaRule(zerolog.New(os.Stderr), ctx, &sppb.Rule{
				Label: tc.from.String(),
				Kind:  tc.rule.Kind(),
				Files: tc.files,
			})

			got := scalaRule.Provides()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaRuleImports(t *testing.T) {
	for name, tc := range map[string]struct {
		directives    []string
		rule          *rule.Rule
		from          label.Label
		files         []*sppb.File
		globalSymbols []*resolver.Symbol // list of symbols in global scope
		want          []string
	}{
		"degenerate": {
			rule: rule.NewRule("scala_library", "somelib"), // rule must not be nil
			want: []string{},
		},
		"explicit imports + extends": {
			globalSymbols: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "com.foo.ClassA",
					Provider: "source",
					Label:    label.Label{Pkg: "com/foo", Name: "somelib"},
				},
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "akka.actor.Actor",
					Provider: "maven",
					Label:    label.Label{Repo: "maven", Name: "akka_actor_akka_actor"},
				},
			},
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Imports:  []string{"com.foo.Bar"},
					Classes:  []string{"com.foo.ClassA", "com.foo.ClassB"},
					Extends: map[string]*sppb.ClassList{
						"class com.foo.ClassA": {Classes: []string{"akka.actor.Actor"}},
						"class com.foo.ClassB": {Classes: []string{"com.foo.ClassA"}},
					},
				},
			},
			want: []string{
				"✅ com.foo.Bar<> (DIRECT of A.scala)",
				`✅ akka.actor.Actor<CLASS> @maven//:akka_actor_akka_actor<maven> (EXTENDS of A.scala via "com.foo.ClassA")`,
			},
		},
		"extends symbol completed by wildcard import": {
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			globalSymbols: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "akka.actor.Actor",
					Label:    label.Label{Repo: "maven", Name: "akka_actor_akka_actor"},
					Provider: "maven",
				},
			},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Imports:  []string{"akka.actor._"},
					Classes:  []string{"com.foo.ClassA"},
					Extends: map[string]*sppb.ClassList{
						"class com.foo.ClassA": {Classes: []string{"Actor"}},
					},
				},
			},
			want: []string{
				"✅ akka.actor._<> (DIRECT of A.scala)",
				`✅ akka.actor.Actor<CLASS> @maven//:akka_actor_akka_actor<maven> (EXTENDS of A.scala via "com.foo.ClassA")`,
			},
		},
		"resolve_with via extends": {
			directives: []string{
				"resolve_with scala com.typesafe.scalalogging.LazyLogging org.slf4j.Logger",
			},
			globalSymbols: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "com.typesafe.scalalogging.LazyLogging",
					Label:    label.Label{Repo: "maven", Name: "com_typesafe_scalalogging"},
					Provider: "maven",
				},
			},
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Classes:  []string{"com.foo.ClassA"},
					Extends: map[string]*sppb.ClassList{
						"class com.foo.ClassA": {Classes: []string{"com.typesafe.scalalogging.LazyLogging"}},
					},
				},
			},
			want: []string{
				`✅ com.typesafe.scalalogging.LazyLogging<CLASS> @maven//:com_typesafe_scalalogging<maven> (EXTENDS of A.scala via "com.foo.ClassA")`,
				`✅ org.slf4j.Logger<> (IMPLICIT via "com.typesafe.scalalogging.LazyLogging")`,
			},
		},
		"resolve_with self type": {
			directives: []string{
				"resolve_with scala com.foo.ClassA com.foo.ClassB",
			},
			globalSymbols: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "com.foo.ClassA",
					Provider: "source",
					Label:    label.Label{Pkg: "com/foo", Name: "somelib"},
				},
			},
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Classes:  []string{"com.foo.ClassA"},
				},
			},
			want: []string{
				`✅ com.foo.ClassB<> (IMPLICIT via "com.foo.ClassA")`,
			},
		},
		"transitive require - this is done later": {
			globalSymbols: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "com.foo.proto.FooMessage",
					Provider: "protobuf",
					Label:    label.Label{Pkg: "proto", Name: "foo_proto_scala_library"},
					Requires: []*resolver.Symbol{
						{
							Type:     sppb.ImportType_CLASS,
							Name:     "scalapb.GeneratedMessage",
							Provider: "java",
							Label:    label.Label{Repo: "maven", Name: "scalapb_runtime"},
						},
					},
				},
			},
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Imports:  []string{"com.foo.proto._"},
					Classes:  []string{"com.foo.ClassA"},
					Extends: map[string]*sppb.ClassList{
						"class com.foo.ClassA": {Classes: []string{"FooMessage"}},
					},
				},
			},
			want: []string{
				"✅ com.foo.proto._<> (DIRECT of A.scala)",
				`✅ com.foo.proto.FooMessage<CLASS> //proto:foo_proto_scala_library<protobuf> (EXTENDS of A.scala via "com.foo.ClassA")`,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			universe := newMockGlobalScope(t, tc.globalSymbols)

			global := resolver.NewTrieScope()
			for _, symbol := range tc.globalSymbols {
				global.PutSymbol(symbol)
			}

			sc, err := NewTestScalaConfig(t, universe, tc.from.Pkg, makeDirectives(tc.directives)...)
			if err != nil {
				t.Fatal(err)
			}

			ctx := &scalaRuleContext{
				rule:        tc.rule,
				scalaConfig: sc,
				resolver:    universe,
				scope:       universe,
			}

			scalaRule := newScalaRule(zerolog.New(os.Stderr), ctx, &sppb.Rule{
				Label: tc.from.String(),
				Kind:  tc.rule.Kind(),
				Files: tc.files,
			})

			imports := scalaRule.Imports(tc.from)
			got := make([]string, len(imports.Keys()))
			for i, imp := range imports.Values() {
				got[i] = imp.String()
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func makeDirectives(in []string) (out []rule.Directive) {
	for _, s := range in {
		if s == "" {
			continue
		}
		fields := strings.Fields(s)
		out = append(out, rule.Directive{Key: fields[0], Value: strings.Join(fields[1:], " ")})
	}
	return
}

type mockGlobalScope struct {
	resolver.Universe
	Global resolver.Scope
}

func newMockGlobalScope(t *testing.T, known []*resolver.Symbol) *mockGlobalScope {
	scope := &mockGlobalScope{
		Universe: mocks.NewUniverse(t),
		Global:   resolver.NewTrieScope(),
	}
	for _, symbol := range known {
		scope.Global.PutSymbol(symbol)
	}
	return scope
}

// GetScope returns a scope for th symbol under the given prefix.
func (m *mockGlobalScope) GetScope(name string) (resolver.Scope, bool) {
	return m.Global.GetScope(name)
}

// GetSymbol does a lookup of the given import symbol and returns the
// known import.  If not known `(nil, false)` is returned.
func (m *mockGlobalScope) GetSymbol(name string) (*resolver.Symbol, bool) {
	return m.Global.GetSymbol(name)
}

// GetSymbols does a lookup of the given prefix and returns the
// symbols.
func (m *mockGlobalScope) GetSymbols(prefix string) []*resolver.Symbol {
	return m.Global.GetSymbols(prefix)
}

func NewTestScalaConfig(t *testing.T, universe resolver.Universe, rel string, dd ...rule.Directive) (*scalaconfig.Config, error) {
	c := config.New()
	sc := scalaconfig.New(zerolog.New(os.Stderr), universe, c, rel)
	err := sc.ParseDirectives(dd)
	return sc, err
}
