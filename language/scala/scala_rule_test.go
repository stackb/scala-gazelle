package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stretchr/testify/mock"
)

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
		// "degenerate": {},
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
			if len(tc.want) > 0 {
				capture := func(known *resolver.KnownImport) bool {
					got = append(got, known)
					return true
				}
				knownImportRegistry.
					On("PutKnownImport", mock.MatchedBy(capture)).
					Times(len(tc.want)).
					Return(nil)
			}

			NewScalaRule(knownImportRegistry, knownImportResolver, tc.rule, tc.from, tc.files)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
