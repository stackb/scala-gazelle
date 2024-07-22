package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"

	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/stackb/scala-gazelle/pkg/scalarule"
)

func TestScalaPackageParseRule(t *testing.T) {

	for name, tc := range map[string]struct {
		rule     *rule.Rule
		attrName string
		want     scalarule.Rule
		wantErr  string
	}{
		"degenerate": {
			rule:    rule.NewRule("scala_library", "somelib"),
			wantErr: "rule has no source files",
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
			cfg := scalaconfig.New(universe, c, "")

			pkg := scalaPackage{
				cfg: cfg,
			}

			var gotErr string
			got, gotError := pkg.ParseRule(tc.rule, tc.attrName)
			if gotError != nil {
				gotErr = gotError.Error()
			}

			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
