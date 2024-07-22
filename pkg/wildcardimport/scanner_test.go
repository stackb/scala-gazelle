package wildcardimport

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
			want: []string{
				"ZonedDateTime",
			},
		},
		"example 2": {
			output: `
ERROR: /Users/pcj/go/src/github.com/Omnistac/unity/omnistac/euds/common/masking/BUILD.bazel:3:21: scala @//omnistac/euds/common/masking:scala failed: (Exit 1): scalac failed: error executing command (from target //omnistac/euds/common/masking:scala) bazel-out/darwin_arm64-opt-exec-2B5CBBC6/bin/external/io_bazel_rules_scala/src/java/io/bazel/rulesscala/scalac/scalac '--jvm_flag=-Xss32M' ... (remaining 1 argument skipped)
omnistac/euds/common/masking/MaskingFacade.scala:11: error: [rewritten by -quickfix] not found: type UserContext
  def maskBlotterOrderEvent(event: BlotterOrderEvent, userContext: Option[UserContext]): Option[BlotterOrderEvent] = {
                                                                          ^
omnistac/euds/common/masking/MaskingFacade.scala:70: error: [rewritten by -quickfix] not found: type UserContext
      userContext: Option[UserContext]
                          ^
omnistac/euds/common/masking/MaskingFacade.scala:23: error: [rewritten by -quickfix] not found: type UserContext
      userContext: Option[UserContext]
                          ^
omnistac/euds/common/masking/MaskingFacade.scala:26: error: [rewritten by -quickfix] not found: type TrumidUserContext
    def ignoreTrumidUser(trumidUser: TrumidUserContext): Boolean =
                                     ^
omnistac/euds/common/masking/MaskingFacade.scala:33: error: [rewritten by -quickfix] not found: type TrumidUserContext
      case trumidUser: TrumidUserContext if !ignoreTrumidUser(trumidUser) =>
                       ^
omnistac/euds/common/masking/MaskingFacade.scala:61: error: [rewritten by -quickfix] not found: type CounterpartyUserContext
      case cptyUser: CounterpartyUserContext if orderEvent.getCptyId == cptyUser.counterpartyId      => orderEvent
                     ^
omnistac/euds/common/masking/MaskingFacade.scala:62: error: [rewritten by -quickfix] not found: type TradingFirmUserContext
      case tradingFirmUser: TradingFirmUserContext if orderEvent.getFirmId == tradingFirmUser.firmId => orderEvent
                            ^
omnistac/euds/common/masking/MaskingFacade.scala:63: error: [rewritten by -quickfix] not found: type TradingAccountUserContext
      case tradingAccountUser: TradingAccountUserContext if orderEvent.getAccountId == tradingAccountUser.accountId =>
                               ^
omnistac/euds/common/masking/MaskingFacade.scala:73: error: [rewritten by -quickfix] not found: type TrumidUserContext
      case trumidUser: TrumidUserContext =>
                       ^
omnistac/euds/common/masking/MaskingFacade.scala:107: error: [rewritten by -quickfix] not found: type CounterpartyUserContext
      case cptyUser: CounterpartyUserContext if stagedIoiEvent.getCptyId == cptyUser.counterpartyId => stagedIoiEvent
                     ^
omnistac/euds/common/masking/MaskingFacade.scala:109: error: [rewritten by -quickfix] not found: type TradingFirmUserContext
      case tradingFirmUser: TradingFirmUserContext if stagedIoiEvent.getFirmId == tradingFirmUser.firmId =>
                            ^
omnistac/euds/common/masking/MaskingFacade.scala:112: error: [rewritten by -quickfix] not found: type TradingAccountUserContext
      case tradingAccountUser: TradingAccountUserContext
                               ^
12 errors
Build failed
java.lang.RuntimeException: Build failed
	at io.bazel.rulesscala.scalac.ScalacInvoker.invokeCompiler(ScalacInvoker.java:55)
	at io.bazel.rulesscala.scalac.ScalacWorker.compileScalaSources(ScalacWorker.java:253)
	at io.bazel.rulesscala.scalac.ScalacWorker.work(ScalacWorker.java:69)
	at io.bazel.rulesscala.worker.Worker.persistentWorkerMain(Worker.java:86)
	at io.bazel.rulesscala.worker.Worker.workerMain(Worker.java:39)
	at io.bazel.rulesscala.scalac.ScalacWorker.main(ScalacWorker.java:33)
`,
			want: []string{
				"CounterpartyUserContext",
				"TradingAccountUserContext",
				"TradingFirmUserContext",
				"TrumidUserContext",
				"UserContext",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := scanOutputForNotFoundSymbols([]byte(tc.output))
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
