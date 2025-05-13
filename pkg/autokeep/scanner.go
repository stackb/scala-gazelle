package autokeep

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	akpb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/autokeep"
)

// ERROR: /Users/pcj/go/src/github.com/Omnistac/unity/omnistac/gum/dao/BUILD.bazel:1550:21: scala @//omnistac/gum/dao:trading_account_anonymous_config_dao_scala failed: (Exit 1): scalac failed: error executing command (from target //omnistac/gum/dao:trading_account_anonymous_config_dao_scala) bazel-out/darwin_arm64-opt-exec-2B5CBBC6/bin/external/io_bazel_rules_scala/src/java/io/bazel/rulesscala/scalac/scalac '--jvm_flag=-Xss32M' ... (remaining 1 argument skipped)
var scalacErrorLine = regexp.MustCompile(`^ERROR: ([^:]+):.*scalac failed.*\(from target ([^\)]+)\).*$`)

// omnistac/gum/dao/TradingAccountAnonymousConfigDao.scala:150: error: Symbol 'type omnistac.gum.entity.TradingAccountDbRecord' is missing from the classpath.
var missingSymbolLine = regexp.MustCompile(`^(.*):\d+: error: Symbol 'type ([^']+)' is missing from the classpath.$`)

// omnistac/postswarm/src/it/scala/omnistac/postswarm/grey/SelectiveSpottingTest.scala:22: error: [rewritten by -quickfix] object SelectiveSpotSessionUtils is not a member of package omnistac.postswarm
var notAMemberOfPackageLine = regexp.MustCompile(`^(.*):\d+: error: .* object ([A-Z][_a-zA-Z0-9]*) is not a member of package (.*)$`)

// trumid/fix/common/testing/src/QuickfixTestUtils.scala:88: error: [rewritten by -quickfix] not found: type SimpleQuickfixSessionObject
var typeNotFoundLine = regexp.MustCompile(`^(.*):\d+: error: .* not found: type ([A-Z][_a-zA-Z0-9]*)$`)

// This symbol is required by 'class omnistac.gum.dao.TradingAccountDao.TradingAccountTable'.
var symbolRequiredByLine = regexp.MustCompile(`^This symbol is required by '([^']+)'.$`)

// buildozer 'remove deps //omnistac/core/biz/validator/ordervalidation:stubs' //omnistac/postswarm:listtrading_perf
var buildozerLine = regexp.MustCompile(`^buildozer 'remove deps ([^']+)' (.*)$`)

func ScanOutput(output []byte) (*akpb.Diagnostics, error) {
	diagnostics := new(akpb.Diagnostics)

	// scalacError is populated when we hit the first scalacErrorLine and
	// becomes a contextual object upon which the state of future errors (lines
	// following) depends.
	var scalacError *akpb.ScalacError

	addError := func(sce *akpb.ScalacError) {
		sce.BuildFile = scalacError.BuildFile
		sce.RuleLabel = scalacError.RuleLabel
		diagnostics.ScalacErrors = append(diagnostics.ScalacErrors, sce)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fmt.Fprintln(os.Stderr, line)
		if line == "" {
			continue
		}

		if match := scalacErrorLine.FindStringSubmatch(line); match != nil {
			scalacError = new(akpb.ScalacError)
			scalacError.BuildFile = match[1]
			scalacError.RuleLabel = strings.TrimSuffix(match[2], "_testlib")
		} else if match := symbolRequiredByLine.FindStringSubmatch(line); match != nil {
			if len(diagnostics.ScalacErrors) > 0 {
				lastError := diagnostics.ScalacErrors[len(diagnostics.ScalacErrors)-1]
				if e, ok := lastError.Error.(*akpb.ScalacError_MissingSymbol); ok {
					e.MissingSymbol.RequiredBy = match[1]
				}
			}
		} else if match := missingSymbolLine.FindStringSubmatch(line); match != nil {
			addError(&akpb.ScalacError{
				Error: &akpb.ScalacError_MissingSymbol{
					MissingSymbol: &akpb.MissingSymbol{
						SourceFile: match[1],
						Symbol:     match[2],
					},
				},
			})
		} else if match := typeNotFoundLine.FindStringSubmatch(line); match != nil {
			addError(&akpb.ScalacError{
				Error: &akpb.ScalacError_NotFound{
					NotFound: &akpb.TypeNotFound{
						SourceFile: match[1],
						Type:       match[2],
					},
				},
			})
		} else if match := notAMemberOfPackageLine.FindStringSubmatch(line); match != nil {
			addError(&akpb.ScalacError{
				Error: &akpb.ScalacError_NotAMemberOfPackage{
					NotAMemberOfPackage: &akpb.NotAMemberOfPackage{
						SourceFile:  match[1],
						Symbol:      match[2],
						PackageName: match[3],
					},
				},
			})
		} else if match := buildozerLine.FindStringSubmatch(line); match != nil {
			addError(&akpb.ScalacError{
				Error: &akpb.ScalacError_BuildozerUnusedDep{
					BuildozerUnusedDep: &akpb.BuildozerUnusedDep{
						UnusedDep: match[1],
						RuleLabel: match[2],
					},
				},
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return diagnostics, nil
}
