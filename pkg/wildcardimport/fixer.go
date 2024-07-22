package wildcardimport

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

type FixerOptions struct {
	BazelExecutable string
}

type Fixer struct {
	bazelExe string
}

func NewFixer(options *FixerOptions) *Fixer {
	bazelExe := options.BazelExecutable
	if bazelExe == "" {
		bazelExe = "bazel"
	}

	return &Fixer{
		bazelExe: bazelExe,
	}
}

// Fix uses iterative bazel builds to remove wildcard imports and returns a list
// of unqualified symbols that were used to complete the import.
func (w *Fixer) Fix(ruleLabel, filename, importPrefix string) ([]string, error) {
	targetLine := fmt.Sprintf("import %s._", importPrefix)

	tf, err := NewTextFileFromFilename(filename, targetLine)
	if err != nil {
		return nil, err
	}
	symbols, err := w.fixFile(ruleLabel, tf, importPrefix)
	if err != nil {
		return nil, err
	}
	return symbols, nil
}

func (w *Fixer) fixFile(ruleLabel string, tf *TextFile, importPrefix string) ([]string, error) {

	// the complete list of not found symbols
	allNotFound := make(map[string]bool)

	// on each build, parse the output for notFound symbols.  Stop the loop when
	// the output is the same as the previous one (nothing more actionable).
	previouslyNotFound := []string{}

	// initialize the scanner
	scanner := &outputScanner{}

	var iteration int
	for {
		if iteration == 0 {
			// rewrite the file clean on the first iteration
			if err := tf.WriteClean(); err != nil {
				return nil, err
			}
		} else if iteration == 1 {
			// comment out the target line on the 2nd iteration
			if err := tf.WriteCommented(); err != nil {
				return nil, err
			}
		}

		// execute the build and gather output
		output, exitCode, cmdErr := execBazelBuild(w.bazelExe, ruleLabel)

		// must build clean first time
		if iteration == 0 && exitCode != 0 {
			return nil, fmt.Errorf("%v: target must build first time: %v (%v)", ruleLabel, string(output), cmdErr)
		}

		// on subsequent iterations if the exitCode is 0, the process is successful.
		if exitCode == 0 {
			if iteration == 0 {
				iteration++
				continue
			}
			// SUCCESS!
			// TODO(pcj): format multiline if too long
			symbols := make([]string, 0, len(allNotFound))
			for sym := range allNotFound {
				symbols = append(symbols, sym)
			}
			return symbols, nil
		}

		log.Printf(">>> fixing %s [%s] (iteration %d)\n", tf.filename, importPrefix, iteration)
		log.Println(">>>", string(output), cmdErr)

		// scan the output for symbols that were not found
		notFounds, err := scanner.scan(output)
		if err != nil {
			return nil, fmt.Errorf("scanning output: %w", err)
		}

		// if no notFound symbols were found, the process failed, but we have
		// nothing actionable.
		if reflect.DeepEqual(previouslyNotFound, notFounds) {
			return nil, fmt.Errorf("expand wildcard failed: final set of notFound symbols: %v", notFounds)
		}

		// rewrite the file with the updated import (and continue)
		if err := tf.Write(makeImportLine(importPrefix, notFounds)); err != nil {
			return nil, fmt.Errorf("failed to write split file: %v", err)
		}

		previouslyNotFound = notFounds
		for _, sym := range notFounds {
			allNotFound[sym] = true
		}
		iteration++
	}
}

func makeImportLine(importPrefix string, symbols []string) string {
	return fmt.Sprintf("import %s.{%s}", importPrefix, strings.Join(symbols, ", "))
}
