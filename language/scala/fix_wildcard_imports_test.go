package scala

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScanOutputForNotFound(t *testing.T) {
	for name, tc := range map[string]struct {
		output  string
		wantErr string
		want    []string
	}{
		"degenerate": {
			want: []string{},
		},
		"example 1": {
			output: `
ERROR: /Users/pcj/go/src/github.com/Omnistac/unity/omnistac/gum/dao/BUILD.bazel:48:21: scala @//omnistac/gum/dao:auth_dao_scala failed: (Exit 1): scalac failed: error executing command (from target //omnistac/gum/dao:auth_dao_scala) bazel-out/darwin_arm64-opt-exec-2B5CBBC6/bin/external/io_bazel_rules_scala/src/java/io/bazel/rulesscala/scalac/scalac '--jvm_flag=-Xss32M' ... (remaining 1 argument skipped)
omnistac/gum/dao/AuthDao.scala:15: error: [rewritten by -quickfix] not found: type ZonedDateTime
    passwordLastUpdatedTimestamp: ZonedDateTime = ZonedDateTime.now(DateUtils.SYSTEM_TZ),
                                  ^
omnistac/gum/dao/AuthDao.scala:15: error: [rewritten by -quickfix] not found: value ZonedDateTime
    passwordLastUpdatedTimestamp: ZonedDateTime = ZonedDateTime.now(DateUtils.SYSTEM_TZ),
                                                  ^
omnistac/gum/dao/AuthDao.scala:32: error: [rewritten by -quickfix] not found: type ZonedDateTime
  def putNewPasswordToken(userId: ActorId, token: String, expirationTs: ZonedDateTime): Future[ResponseStatus]
                                                                        ^
3 errors
Build failed
java.lang.RuntimeException: Build failed
	at io.bazel.rulesscala.scalac.ScalacWorker.compileScalaSources(ScalacWorker.java:324)
	at io.bazel.rulesscala.scalac.ScalacWorker.work(ScalacWorker.java:72)
	at io.bazel.rulesscala.worker.Worker.persistentWorkerMain(Worker.java:86)
	at io.bazel.rulesscala.worker.Worker.workerMain(Worker.java:39)
	at io.bazel.rulesscala.scalac.ScalacWorker.main(ScalacWorker.java:36)
Target //omnistac/gum/dao:auth_dao_scala failed to build
Use --verbose_failures to see the command lines of failed build steps.
INFO: Elapsed time: 2.354s, Critical Path: 1.40s
INFO: 2 processes: 2 internal.
FAILED: Build did NOT complete successfully
`,
			want: []string{"ZonedDateTime"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := scanOutputForNotFound([]byte(tc.output))
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("result (-want +got):\n%s", diff)
			}
		})
	}
}
