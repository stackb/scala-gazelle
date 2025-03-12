package wildcardimport

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

const debug = false

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

	log.Printf("[🚧 fixing...][%s](%s): %s", ruleLabel, filename, targetLine)
	tf, err := NewTextFileFromFilename(filename, targetLine)
	if err != nil {
		if _, isFoundFoundError := err.(*ImportLineNotFoundError); isFoundFoundError {
			log.Printf("WARN: %v", err)
			return nil, nil
		}
		return nil, err
	}
	// if textfile is nil, could not find text pattern.  Move on.
	if tf == nil {
		return nil, nil
	}

	symbols, err := w.fixFile(ruleLabel, tf, importPrefix)
	if err != nil {
		return nil, err
	}

	log.Printf("[✅ fixed][%s](%s): %v", ruleLabel, filename, symbols)

	return symbols, nil
}

func (w *Fixer) fixFile(ruleLabel string, tf *TextFile, importPrefix string) ([]string, error) {

	// the complete list of not found symbols
	completion := map[string]bool{}

	// initialize the scanner
	scanner := &outputScanner{}

	var iteration int
	for {
		if iteration == 0 {
			// rewrite the file clean on the first iteration, in case the
			// previous run edited it.
			if err := tf.WriteOriginal(); err != nil {
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
		if iteration == 0 {
			if exitCode != 0 {
				return nil, fmt.Errorf("%v: target must build first time: %v (%v)", ruleLabel, string(output), cmdErr)
			} else {
				iteration++
				continue
			}
		}

		// on subsequent iterations if the exitCode is 0, the process is
		// successful.
		if exitCode == 0 {
			keys := mapKeys(completion)
			return keys, nil
		}

		if debug {
			log.Printf(">>> fixing %s [%s] (iteration %d)\n", tf.filename, importPrefix, iteration)
			log.Println(">>>", string(output), cmdErr)
		}

		// scan the output for symbols that were not found
		symbols, err := scanner.scan(output)
		if err != nil {
			return nil, fmt.Errorf("scanning output: %w", err)
		}

		if debug {
			log.Printf("iteration %d symbols: %v", iteration, symbols)
		}

		var hasNewResult bool
		for _, sym := range symbols {
			if _, ok := completion[sym]; !ok {
				completion[sym] = true
				hasNewResult = true
			}
		}

		// if no notFound symbols were found, the process failed, but we have
		// nothing actionable.
		if !hasNewResult {
			return nil, fmt.Errorf("expand wildcard failed: final set of notFound symbols: %v", mapKeys(completion))
		}

		impLine, err := makeImportLine(importPrefix, mapKeys(completion))
		if err != nil {
			return nil, err
		}

		// rewrite the file with the updated import (and continue)
		if err := tf.Write(impLine); err != nil {
			return nil, fmt.Errorf("failed to write split file: %v", err)
		}

		iteration++
	}
}

func makeImportLine(importPrefix string, symbols []string) (string, error) {
	if importPrefix == "" {
		return "", fmt.Errorf("importPrefix must not be empty")
	}
	if len(symbols) == 0 {
		return "", fmt.Errorf("must have at least one symbol in list")
	}
	if len(symbols) == 1 {
		return fmt.Sprintf("import %s.%s", importPrefix, symbols[0]), nil
	}
	sort.Strings(symbols)
	return fmt.Sprintf("import %s.{%s}", importPrefix, strings.Join(symbols, ", ")), nil
}

// mapKeys sorts the list of map keys
func mapKeys(in map[string]bool) (out []string) {
	if len(in) == 0 {
		return nil
	}
	for k := range in {
		out = append(out, k)
	}
	return
}
