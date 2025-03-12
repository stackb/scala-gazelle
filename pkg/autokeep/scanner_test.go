package autokeep

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	akpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/autokeep"
)

func TestScanOutput(t *testing.T) {
	for name, tc := range map[string]struct {
		input   string
		want    *akpb.Diagnostics
		wantErr string
	}{
		"degenerate": {
			want: &akpb.Diagnostics{},
		},
		"notFound1": {
			input: `
INFO: Invocation ID: 23b9f0d7-f585-46a7-9592-041259bc2b69
Loading: 
Loading: 
Loading: 0 packages loaded
Analyzing: target //omnistac/postswarm:grey_it (0 packages loaded, 0 targets configured)
INFO: Analyzed target //omnistac/postswarm:grey_it (0 packages loaded, 0 targets configured).
INFO: Found 1 target...
[0 / 3] [Prepa] BazelWorkspaceStatusAction stable-status.txt ... (3 actions, 0 running)
[1 / 3] scala @//omnistac/postswarm:grey_it_testlib; 1s remote-cache, worker ... (4 actions running)
ERROR: /Users/pcj/go/src/github.com/Omnistac/unity/omnistac/postswarm/BUILD.bazel:1000:18: scala @//omnistac/postswarm:grey_it_testlib failed: (Exit 1): scalac failed: error executing command (from target //omnistac/postswarm:grey_it_testlib) bazel-out/darwin_arm64-opt-exec-C7777A24/bin/external/io_bazel_rules_scala/src/java/io/bazel/rulesscala/scalac/scalac '--jvm_flag=-Xss32M' '--jvm_flag=-Djava.security.manager=allow' ... (remaining 1 argument skipped)
omnistac/postswarm/src/it/scala/omnistac/postswarm/grey/SelectiveSpottingTest.scala:22: error: [rewritten by -quickfix] object SelectiveSpotSessionUtils is not a member of package omnistac.postswarm
import omnistac.postswarm.{PostSwarmUtils, SelectiveSpotSessionUtils}
       ^
omnistac/postswarm/src/it/scala/omnistac/postswarm/grey/SelectiveSpottingTest.scala:1597: error: [rewritten by -quickfix] not found: value SelectiveSpotSessionUtils
      SelectiveSpotSessionUtils.ResetImbalanceGroupName)
      ^
`,
			want: &akpb.Diagnostics{
				ScalacErrors: []*akpb.ScalacError{
					{
						RuleLabel: "//omnistac/postswarm:grey_it",
						BuildFile: "/Users/pcj/go/src/github.com/Omnistac/unity/omnistac/postswarm/BUILD.bazel",
						Error: &akpb.ScalacError_NotAMemberOfPackage{
							NotAMemberOfPackage: &akpb.NotAMemberOfPackage{
								Symbol:      "SelectiveSpotSessionUtils",
								PackageName: "omnistac.postswarm",
							},
						},
					},
				},
			},
		},
		"buildozer": {
			input: `
ERROR: /Users/pcj/go/src/github.com/Omnistac/unity/omnistac/microswarm/BUILD.bazel:205:14: scala @//omnistac/microswarm:testing failed: (Exit 1): scalac failed: error executing command (from target //omnistac/microswarm:testing) bazel-out/darwin_arm64-opt-exec-C7777A24/bin/external/io_bazel_rules_scala/src/java/io/bazel/rulesscala/scalac/scalac '--jvm_flag=-Djava.security.manager=allow' ... (remaining 1 argument skipped)
warning: 1 deprecation (since 2.13.0)
warning: 2 deprecations (since 2025-01-01)
warning: 3 deprecations in total; re-run with -deprecation for details
3 warnings
error: Unused dependencies:
error: Target '//omnistac/spok/message:trades_query_request_proto_scala_library' (via jar: ' bazel-out/darwin_arm64-fastbuild/bin/omnistac/spok/message/trades_query_request_proto_scala_library_java.jar ')  is specified as a dependency to //omnistac/microswarm:testing but isn't used, please remove it from the deps.
You can use the following buildozer command:
buildozer 'remove deps //omnistac/spok/message:trades_query_request_proto_scala_library' //omnistac/microswarm:testing
Build failed			
`,
			want: &akpb.Diagnostics{
				ScalacErrors: []*akpb.ScalacError{
					{
						RuleLabel: "//omnistac/microswarm:testing",
						BuildFile: "/Users/pcj/go/src/github.com/Omnistac/unity/omnistac/microswarm/BUILD.bazel",
						Error: &akpb.ScalacError_BuildozerUnusedDep{
							BuildozerUnusedDep: &akpb.BuildozerUnusedDep{
								RuleLabel: "//omnistac/microswarm:testing",
								UnusedDep: "//omnistac/spok/message:trades_query_request_proto_scala_library",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := ScanOutput([]byte(tc.input))
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreUnexported(
				akpb.Diagnostics{},
				akpb.ScalacError{},
				akpb.NotAMemberOfPackage{},
				akpb.BuildozerUnusedDep{},
			)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
