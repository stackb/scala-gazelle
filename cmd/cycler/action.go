package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

var (
	// trumid/common/auth/scala/test/AuthHeadersTest.scala:8: error: not found: type AuthHeaders
	notFoundRe = regexp.MustCompile(`^([^:]+):(\d+): error: not found: (type|value|object) (.*)$`)
	// omnistac/reporter/src/main/scala/omnistac/reporter/blackrockaxesreport/BlackRockAxesExcelReport.scala:23: error: Class org.apache.poi.ss.usermodel.Workbook not found - continuing with a stub.
	classNotFoundRe = regexp.MustCompile(`^([^:]+):(\d+): error: Class (.*) not found - continuing with a stub\.$`)
	symbolMissingRe = regexp.MustCompile(`^([^:]+):(\d+): error: Symbol '(.*)' is missing from the classpath\.$`)
	// error: Symbol 'type omnistac.spok.message.FinraReportingAssetClass' is missing from the classpath.
	symbolMissingNoFileRe = regexp.MustCompile(`^error: Symbol '(.*)' is missing from the classpath\.$`)
	symbolRequiredByRe    = regexp.MustCompile(`^This symbol is required by '(.*)'\.$`)
)

// File represents a single "File" in a bep.json file, or at least the minimal
// structure required for our purposes.  The URI is expected to be a file://
// URI.
type File struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// Action represents a single "ActionExecuted" event in a bep.json file, or
// at least the minimal structure required for our purposes.
type Action struct {
	Label    string `json:"label,omitempty"`
	ExitCode int    `json:"exitCode,omitempty"`
	Stderr   File   `json:"stderr,omitempty"`
}

// parseStderr parses the action stderr file and collects a list of migrations.
func (a *Action) parseStderr(env *env) (migrations []Migration, err error) {
	from, err := label.Parse(a.Label)
	if err != nil {
		return nil, fmt.Errorf("%v: label parse error: %v", a.Label, err)
	}

	if to, ok := env.cfg.labelMappings[from]; ok {
		from = to
	}

	uri, err := url.Parse(a.Stderr.URI)
	if err != nil {
		return nil, fmt.Errorf("%v: url parse error: %v", from, err)
	}

	f, err := os.Open(uri.Path)
	if err != nil {
		log.Printf("action stderr file: %+v", a.Stderr)
		return nil, fmt.Errorf("%v: open stderr file error: %v", a.Label, err)
	}
	defer f.Close()

	// seen is a set of strings.  The keys are migration IDs
	seen := make(map[string]bool)

	addMigration := func(m Migration) {
		id := m.ID()
		if !seen[id] {
			migrations = append(migrations, m)
			seen[id] = true
		}
	}

	// addMissingSymbol adds it to the queue but checks it looks valid before doing so.
	addMissingSymbol := func(m *missingSymbol) error {
		if m.RequiredType == "" {
			return fmt.Errorf("%s:%s: incomplete match (no .RequiredType): %q", m.Filename, m.Line, m.MissingType)
		}
		addMigration(m)
		return nil
	}

	var currentMissingSymbol *missingSymbol

	log.Println("--- SCAN:", uri.Path)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		log.Println("LINE:", line)

		if match := notFoundRe.FindStringSubmatch(line); match != nil {
			m := &notFoundSymbol{
				From:         from,
				Filename:     match[1],
				Line:         match[2],
				NotFoundType: match[4],
			}
			addMigration(m)
			log.Printf("Matched NotFoundSymbol: %q", m.NotFoundType)
			continue
		}

		if match := classNotFoundRe.FindStringSubmatch(line); match != nil {
			m := &notFoundSymbol{
				From:         from,
				Filename:     match[1],
				Line:         match[2],
				NotFoundType: match[3],
			}
			addMigration(m)
			log.Printf("Matched NotFoundSymbol: %q", m.NotFoundType)
			continue
		}

		if match := symbolMissingRe.FindStringSubmatch(line); match != nil {
			fields := strings.Fields(match[3])
			if len(fields) == 2 {
				// push the existing match to completion
				if currentMissingSymbol != nil {
					if err := addMissingSymbol(currentMissingSymbol); err != nil {
						env.ReportError(err)
					}
				}
				currentMissingSymbol = &missingSymbol{
					From:        from,
					Filename:    match[1],
					Line:        match[2],
					MissingType: fields[1],
				}
				log.Printf("Matched Missing Symbol: %+v", currentMissingSymbol)
			}
			continue
		}

		if match := symbolMissingNoFileRe.FindStringSubmatch(line); match != nil {
			fields := strings.Fields(match[1])
			if len(fields) == 2 {
				// push the existing match to completion
				if currentMissingSymbol != nil {
					if err := addMissingSymbol(currentMissingSymbol); err != nil {
						env.ReportError(err)
					}
				}
				currentMissingSymbol = &missingSymbol{
					From:        from,
					Filename:    "",
					Line:        "",
					MissingType: fields[1],
				}
				log.Printf("Matched Missing Symbol: %+v", currentMissingSymbol)
			}
			continue
		}

		if match := symbolRequiredByRe.FindStringSubmatch(line); match != nil && currentMissingSymbol != nil {
			fields := strings.Fields(match[1])
			// LINE: error: Symbol 'type omnistac.spok.message.FinraReportingAssetClass' is missing from the classpath.
			// LINE: This symbol is required by 'value omnistac.core.util.data.DataUtil.finraReportingAssetClass'.
			// TODO: fix above when filename not reported.

			// this can by 'type FOO' or 'value FOO', or 'lazy value FOO', so just use the last one.
			currentMissingSymbol.RequiredType = fields[len(fields)-1]

			// log.Printf("Matched Symbol Required: %+v", currentMissingSymbol)
			continue
		}

	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// push last
	if currentMissingSymbol != nil {
		if err := addMissingSymbol(currentMissingSymbol); err != nil {
			env.ReportError(err)
		}
	}

	if len(migrations) == 0 {
		err = fmt.Errorf("%v: %s: failed to discover any migration strategies", a.Label, a.Stderr.URI)
	}

	return
}
