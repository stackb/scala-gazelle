package scala

import (
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stretchr/testify/mock"
)

func TestScalaRuleRequiredTypes(t *testing.T) {
	for name, tc := range map[string]struct {
		rule  *rule.Rule
		from  label.Label
		files []*sppb.File
		want  map[string][]string
	}{
		"degenerate": {
			want: map[string][]string{},
		},
		"extends": {
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Imports:  []string{"com.foo.Bar"},
					Classes:  []string{"com.foo.Animal", "com.foo.Dog"},
					Extends: map[string]*sppb.ClassList{
						"class com.foo.Dog": {Classes: []string{"com.foo.Animal"}},
					},
				},
			},
			want: map[string][]string{
				"com.foo.Animal": {"com.foo.Dog"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {

			knownImportRegistry := mocks.NewKnownImportRegistry(t)
			knownImportResolver := mocks.NewKnownImportResolver(t)

			knownImportRegistry.
				On("PutKnownImport", mock.Anything).
				Maybe().
				Return(nil)

			scalaRule := NewScalaRule(knownImportRegistry, knownImportResolver, tc.rule, tc.from, tc.files)

			got := scalaRule.requiredTypes
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

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
			knownImportRegistry := mocks.NewKnownImportRegistry(t)
			knownImportResolver := mocks.NewKnownImportResolver(t)

			knownImportRegistry.
				On("PutKnownImport", mock.Anything).
				Maybe().
				Return(nil)

			scalaRule := NewScalaRule(knownImportRegistry, knownImportResolver, tc.rule, tc.from, tc.files)
			got := scalaRule.Exports()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaRulePutKnownImport(t *testing.T) {
	makeSelfImport := func(typ sppb.ImportType, imp string) *resolver.KnownImport {
		return &resolver.KnownImport{Type: typ, Import: imp, Label: label.NoLabel, Provider: "self-import"}
	}

	for name, tc := range map[string]struct {
		rule  *rule.Rule
		from  label.Label
		files []*sppb.File
		want  []*resolver.KnownImport
	}{
		"degenerate": {},
		"known imports": {
			rule: rule.NewRule("scala_library", "somelib"),
			from: label.Label{Pkg: "com/foo", Name: "somelib"},
			files: []*sppb.File{
				{
					Filename: "A.scala",
					Imports:  []string{"com.foo.Bar"},
					Packages: []string{"com.foo"}, // NOTE: the package does not get advertised as a known import
					Classes:  []string{"com.foo.ClassA", "com.foo.ClassB"},
					Objects:  []string{"com.foo.ObjectA", "com.foo.ObjectB"},
					Traits:   []string{"com.foo.TraitA", "com.foo.TraitB"},
					Types:    []string{"com.foo.TypeA", "com.foo.TypeB"},
					Vals:     []string{"com.foo.ValA", "com.foo.ValB"},
					Names:    []string{"com", "foo"}, // names aren't used
				},
			},
			want: []*resolver.KnownImport{
				makeSelfImport(sppb.ImportType_CLASS, "com.foo.ClassA"),
				makeSelfImport(sppb.ImportType_CLASS, "com.foo.ClassB"),
				makeSelfImport(sppb.ImportType_OBJECT, "com.foo.ObjectA"),
				makeSelfImport(sppb.ImportType_OBJECT, "com.foo.ObjectB"),
				makeSelfImport(sppb.ImportType_TRAIT, "com.foo.TraitA"),
				makeSelfImport(sppb.ImportType_TRAIT, "com.foo.TraitB"),
				makeSelfImport(sppb.ImportType_TYPE, "com.foo.TypeA"),
				makeSelfImport(sppb.ImportType_TYPE, "com.foo.TypeB"),
				makeSelfImport(sppb.ImportType_VALUE, "com.foo.ValA"),
				makeSelfImport(sppb.ImportType_VALUE, "com.foo.ValB"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			knownImportRegistry := mocks.NewKnownImportRegistry(t)
			knownImportResolver := mocks.NewKnownImportResolver(t)

			var got []*resolver.KnownImport
			capture := func(known *resolver.KnownImport) bool {
				got = append(got, known)
				return true
			}
			knownImportRegistry.
				On("PutKnownImport", mock.MatchedBy(capture)).
				Maybe().
				Times(len(tc.want)).
				Return(nil)

			NewScalaRule(knownImportRegistry, knownImportResolver, tc.rule, tc.from, tc.files)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaRuleImports(t *testing.T) {
	for name, tc := range map[string]struct {
		directives []string
		rule       *rule.Rule
		from       label.Label
		files      []*sppb.File
		want       []string
	}{
		"degenerate": {
			rule: rule.NewRule("scala_library", "somelib"), // rule must not be nil
			want: []string{},
		},
		"explicit imports + extends": {
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
				"✅ akka.actor.Actor<> (EXTENDS of com.foo.ClassA)",
				"✅ com.foo.Bar<> (DIRECT of A.scala)",
				"✅ com.foo.ClassA<> (EXTENDS of com.foo.ClassB)",
			},
		},
		"resolve_with via extends": {
			directives: []string{
				"resolve_with scala com.typesafe.scalalogging.LazyLogging org.slf4j.Logger",
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
				"✅ com.typesafe.scalalogging.LazyLogging<> (EXTENDS of com.foo.ClassA)",
				"✅ org.slf4j.Logger<> (IMPLICIT of com.typesafe.scalalogging.LazyLogging)",
			},
		},
		"resolve_with self type": {
			directives: []string{
				"resolve_with scala com.foo.ClassA com.foo.ClassB",
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
				"✅ com.foo.ClassB<> (IMPLICIT of com.foo.ClassA)",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			knownImportRegistry := mocks.NewKnownImportRegistry(t)
			importResolver := mocks.NewImportResolver(t)

			knownImportRegistry.
				On("PutKnownImport", mock.Anything).
				Maybe().
				Return(nil)

			scalaRule := NewScalaRule(knownImportRegistry, importResolver, tc.rule, tc.from, tc.files)
			c := config.New()
			scalaConfig := newScalaConfig(c, tc.from.Pkg, importResolver)
			if err := scalaConfig.parseDirectives(makeDirectives(tc.directives)); err != nil {
				t.Fatal(err)
			}
			imports := scalaRule.Imports(scalaConfig)
			got := make([]string, len(imports))
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
