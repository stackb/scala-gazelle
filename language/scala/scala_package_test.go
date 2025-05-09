package scala

import (
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver/mocks"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
)

func TestScalaPackageNewScalaRule(t *testing.T) {
	for name, tc := range map[string]struct {
		rule      *rule.Rule
		wantFiles []*sppb.File
		wantErr   string
	}{
		"degenerate": {
			rule:      rule.NewRule("scala_library", "somelib"),
			wantErr:   "rule has no source files",
			wantFiles: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			logger := zerolog.New(os.Stderr)
			universe := mocks.NewUniverse(t)
			scope := mocks.NewScope(t)
			scope.
				On("PutSymbol", mock.Anything).
				Maybe().
				Return(nil)

			cfg := scalaconfig.New(
				logger,
				universe,
				config.New(),
				"",
			)

			pkg := scalaPackage{
				cfg:    cfg,
				logger: logger,
			}

			var gotErr string
			got, gotError := pkg.NewScalaRule(tc.rule)
			if gotError != nil {
				gotErr = gotError.Error()
			}

			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantFiles, got.Rule().Files); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
