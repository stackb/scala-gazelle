package autokeep

import (
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	akpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/autokeep"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestMakeDeltaDeps(t *testing.T) {
	for name, tc := range map[string]struct {
		input *akpb.Diagnostics
		deps  DepsMap
		want  *akpb.DeltaDeps
	}{
		"degenerate": {
			input: &akpb.Diagnostics{},
			want:  &akpb.DeltaDeps{},
		},
		"not-a-package": {
			deps: map[string]string{
				"contoso.postswarm.SelectiveSpotSessionUtils": "//contoso/postswarm:selective_spot_session_utils_common_scala",
			},
			input: &akpb.Diagnostics{
				ScalacErrors: []*akpb.ScalacError{
					{
						RuleLabel: "//contoso/postswarm:grey_it",
						BuildFile: "/home/user/src/github.com/contoso/unity/contoso/postswarm/BUILD.bazel",
						Error: &akpb.ScalacError_NotAMemberOfPackage{
							NotAMemberOfPackage: &akpb.NotAMemberOfPackage{
								Symbol:      "SelectiveSpotSessionUtils",
								PackageName: "contoso.postswarm",
							},
						},
					},
				},
			},
			want: &akpb.DeltaDeps{
				Add: []*akpb.RuleDeps{
					{
						Label:     "//contoso/postswarm:grey_it",
						BuildFile: "/home/user/src/github.com/contoso/unity/contoso/postswarm/BUILD.bazel",
						Deps:      []string{"//contoso/postswarm:selective_spot_session_utils_common_scala"},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := MakeDeltaDeps(tc.deps, tc.input)

			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreUnexported(
				akpb.DeltaDeps{},
				akpb.RuleDeps{},
			)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestApplyDeltaDeps(t *testing.T) {
	for name, tc := range map[string]struct {
		DeltaDeps   *akpb.DeltaDeps
		keepComment bool
		files       []testtools.FileSpec
		want        []testtools.FileSpec
		wantErr     string
	}{
		"degenerate": {
			DeltaDeps: &akpb.DeltaDeps{},
			files:     []testtools.FileSpec{},
			want:      []testtools.FileSpec{},
		},
		"adds matching deps": {
			DeltaDeps: &akpb.DeltaDeps{
				Add: []*akpb.RuleDeps{
					{
						Label:     "//contoso/postswarm:tests",
						BuildFile: "src/github.com/org/repo/contoso/postswarm/BUILD.bazel",
						Deps:      []string{"//contoso/postswarm:selective_spot_session_utils_common_scala"},
					},
				},
			},
			files: []testtools.FileSpec{
				{
					Path: "src/github.com/org/repo/contoso/postswarm/BUILD.bazel",
					Content: `
scala_library(
    name = "tests",
	srcs = glob(["*.scala"]),
	deps = [],
)
`,
				},
			},
			want: []testtools.FileSpec{
				{
					Path: "src/github.com/org/repo/contoso/postswarm/BUILD.bazel",
					Content: `scala_library(
    name = "tests",
    srcs = glob(["*.scala"]),
    deps = ["//contoso/postswarm:selective_spot_session_utils_common_scala"],
)
`,
				},
			},
		},
		"removes matching deps": {
			DeltaDeps: &akpb.DeltaDeps{
				Remove: []*akpb.RuleDeps{
					{
						Label:     "//contoso/postswarm:tests",
						BuildFile: "src/github.com/org/repo/contoso/postswarm/BUILD.bazel",
						Deps:      []string{"//contoso/postswarm:selective_spot_session_utils_common_scala"},
					},
				},
			},
			files: []testtools.FileSpec{
				{
					Path: "src/github.com/org/repo/contoso/postswarm/BUILD.bazel",
					Content: `
scala_library(
    name = "tests",
	srcs = glob(["*.scala"]),
	deps = ["//contoso/postswarm:selective_spot_session_utils_common_scala"],
)
`,
				},
			},
			want: []testtools.FileSpec{
				{
					Path: "src/github.com/org/repo/contoso/postswarm/BUILD.bazel",
					Content: `scala_library(
    name = "tests",
    srcs = glob(["*.scala"]),
    deps = [],
)
`,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustPrepareTestFiles(t, tc.files)
			if true {
				defer cleanup()
			}

			// Prepend the tmpDir to file paths since actual real-world usage
			// assumes the files are absolute.
			for _, rule := range tc.DeltaDeps.Add {
				rule.BuildFile = path.Join(tmpDir, rule.BuildFile)
			}
			for _, rule := range tc.DeltaDeps.Remove {
				rule.BuildFile = path.Join(tmpDir, rule.BuildFile)
			}

			err := ApplyDeltaDeps(tc.DeltaDeps, tc.keepComment)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}

			got := make([]testtools.FileSpec, 0, len(tc.files))
			for _, rule := range tc.DeltaDeps.Add {
				rule.BuildFile = strings.TrimPrefix(strings.TrimPrefix(rule.BuildFile, tmpDir), "/")
				got = append(got, testtools.FileSpec{
					Path:    rule.BuildFile,
					Content: testutil.MustReadTestFile(t, tmpDir, rule.BuildFile),
				})
			}
			for _, rule := range tc.DeltaDeps.Remove {
				rule.BuildFile = strings.TrimPrefix(strings.TrimPrefix(rule.BuildFile, tmpDir), "/")
				got = append(got, testtools.FileSpec{
					Path:    rule.BuildFile,
					Content: testutil.MustReadTestFile(t, tmpDir, rule.BuildFile),
				})
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
