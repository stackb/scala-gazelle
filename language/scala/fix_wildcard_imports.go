package scala

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
)

func fixWildcardRuleImports(sc *scalaconfig.Config, rule *sppb.Rule) error {
	if rule.Label != "//omnistac/gum/dao:auth_dao_scala" {
		return nil
	}
	log.Println("fixing wildcards for rule:", rule.Label)
	for _, file := range rule.Files {
		if err := fixWildcardFileImports(sc, rule, file); err != nil {
			return err
		}
	}
	return nil
}

func fixWildcardFileImports(sc *scalaconfig.Config, rule *sppb.Rule, file *sppb.File) error {
	// log.Println("fixing wildcards for file:", file.Filename)
	for _, imp := range file.Imports {
		if err := fixWildcardSingleImport(sc, rule, file, imp); err != nil {
			return err
		}
	}
	return nil
}

func fixWildcardSingleImport(sc *scalaconfig.Config, rule *sppb.Rule, file *sppb.File, imp string) error {
	if _, ok := resolver.IsWildcardImport(imp); !ok {
		return nil
	}
	if strings.HasSuffix(imp, ".api._") {
		return nil
	}
	if strings.HasSuffix(imp, ".Implicits._") {
		return nil
	}

	filename := path.Join(sc.Config().WorkDir, sc.Rel(), file.Filename)
	// log.Println("fixing wildcard for file:", filename, imp)

	targetLine := "import " + imp
	sf, err := sliceFile(filename, targetLine)
	if err != nil {
		// return fmt.Errorf("failed to slice file: %v", err)
		log.Printf("failed to slice file: %v", err)
		return nil
	}

	err = runFixLoop(rule, sf, strings.TrimSuffix(imp, "._"))
	if err == nil {
		return nil
	}
	log.Printf("fix error: %v (will restore original file)", err)

	// something went wrong - restore original file
	if restoreErr := sf.restore(); restoreErr != nil {
		return restoreErr
	}

	return err
}

func runFixLoop(rule *sppb.Rule, sf *slicedFile, importPrefix string) error {

	// on each build, parse the output for notFound symbols.  Stop the loop when
	// the output is the same as the previous one (nothing more actionable).
	previouslyNotFound := []string{}

	var iteration int
	for {
		iteration++

		output, exitCode, err := bazelBuild("bazel", rule.Label)
		if err != nil {
			return err
		}

		log.Printf(">>> [%s][%d]: fixing %s._\n", sf.filename, iteration, importPrefix)
		log.Println(string(output))

		// on the first iteration ensure the target builds so we have a clean
		// slate.  Then, comment out the targetLine (and rebuild...)
		if iteration == 1 {
			if exitCode != 0 {
				return fmt.Errorf("%v: target must build first time: %v", rule.Label, err)
			}
			if err := sf.write(fmt.Sprintf("// import %s._", importPrefix)); err != nil {
				return err
			}
			continue
		}

		// on subsequent iterations if the exitCode is 0, the process is successful.
		if exitCode == 0 {
			return nil
		}

		// scan the output for symbols that were not found
		notFound, err := scanOutputForNotFound(output)
		if err != nil {
			return err
		}

		// if no notFound symbols were found, the process failed, but we have
		// nothing actionable.
		if reflect.DeepEqual(previouslyNotFound, notFound) {
			return fmt.Errorf("expand wildcard failed: final set of notFound symbols: %v", notFound)
		}

		// rewrite the file with the updated import (and continue)
		if err := sf.write(makeImportLine(importPrefix, notFound)); err != nil {
			return fmt.Errorf("failed to write split file: %v", err)
		}
	}
}

func makeImportLine(importPrefix string, symbols []string) string {
	return fmt.Sprintf("import %s.{%s}", importPrefix, strings.Join(symbols, ", "))
}

type slicedFile struct {
	filename string
	info     fs.FileInfo

	beforeLines []string
	targetLine  string
	afterLines  []string
}

func (f *slicedFile) restore() error {
	return f.write(f.targetLine)
}

func (f *slicedFile) write(targetLine string) error {
	lines := append(f.beforeLines, targetLine)
	lines = append(lines, f.afterLines...)
	content := strings.Join(lines, "\n")
	data := []byte(content)
	if err := os.WriteFile(f.filename, data, f.info.Mode()); err != nil {
		return err
	}
	return nil
}

func sliceFile(filename string, targetLine string) (*slicedFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return sliceInput(filename, info, f, targetLine)
}

func sliceInput(filename string, info fs.FileInfo, in io.Reader, targetLine string) (*slicedFile, error) {
	file := new(slicedFile)
	file.filename = filename
	file.info = info

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if line == targetLine {
			file.targetLine = line
			continue
		}
		if file.targetLine == "" {
			file.beforeLines = append(file.beforeLines, line)
		} else {
			file.afterLines = append(file.afterLines, line)
		}
	}
	if file.targetLine == "" {
		return nil, fmt.Errorf("%s: slice target line not found: %v", filename, targetLine)
	}

	// add a final entry to afterLines so that the file ends with a single newline
	file.afterLines = append(file.afterLines, "")

	return file, nil
}

// omnistac/gum/testutils/DbDataInitUtils.scala:98: error: [rewritten by -quickfix] not found: value FixSessionDao
var notFoundLine = regexp.MustCompile(`^(.*):\d+: error: .*not found: (value|type) (.*)$`)

func bazelBuild(bazelExe string, label string) ([]byte, int, error) {
	args := []string{"build", label}

	command := exec.Command(bazelExe, args...)
	command.Dir = getCommandDir()

	log.Println("!!!", command.String())
	output, err := command.CombinedOutput()
	log.Println("cmdErr:", err)
	if err != nil {
		// Check for exit errors specifically
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode := waitStatus.ExitStatus()
			return nil, exitCode, err
		} else {
			return nil, -1, err
		}
	}
	return output, 0, nil
}

func scanOutputForNotFound(output []byte) ([]string, error) {
	notFound := make([]string, 0)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		log.Println("line:", line)
		if match := notFoundLine.FindStringSubmatch(line); match != nil {
			typeOrValue := match[3]
			notFound = append(notFound, typeOrValue)
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return notFound, nil
}

func getCommandDir() string {
	if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
		return bwd
	} else {
		return "."
	}
}
