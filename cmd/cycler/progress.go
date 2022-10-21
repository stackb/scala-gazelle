package main

import (
	"bufio"
	"log"
	"regexp"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var (
	// ERROR: /Users/i868039/go/src/github.com/Omnistac/unity/omnistac/postswarm/BUILD.bazel:177:18: no such target '//trumid/ats/common/proto:scala_proto': target 'scala_proto' not declared in package 'trumid/ats/common/proto' (did you mean 'java_proto'?) defined by /Users/i868039/go/src/github.com/Omnistac/unity/trumid/ats/common/proto/BUILD.bazel and referenced by '//omnistac/postswarm:flaky.it'
	noSuchTargetRe            = regexp.MustCompile(`^ERROR: ([^:]+):(\d+):(\d+): no such target '(.*)': .* and referenced by '(.*)'$`)
	buildozerRecommentationRe = regexp.MustCompile(`^buildozer '(.*)' (.*)$`)
	ansiRe                    = regexp.MustCompile(ansi)
)

// Progress represents a single "Progress" event in a bep.json file.
type Progress struct {
	Stderr string `json:"stderr,omitempty"`
}

// parseStderr parses the action stderr file and collects a list of migrations.
func (p *Progress) parseStderr(env *env) (migrations []Migration, err error) {
	// seen is a set of strings.  The keys are migration IDs.
	seen := make(map[string]bool)

	addMigration := func(m Migration) {
		id := m.ID()
		if !seen[id] {
			migrations = append(migrations, m)
			seen[id] = true
		}
	}

	log.Printf("--- SCAN Progress Stderr (%db)", len(p.Stderr))

	scanner := bufio.NewScanner(strings.NewReader(stripAnsi(p.Stderr)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		log.Println("LINE:", line)

		if match := noSuchTargetRe.FindStringSubmatch(line); match != nil {
			depStr := match[4]
			fromStr := match[5]
			dep, err := label.Parse(depStr)
			if err != nil {
				log.Printf("warning: failed to parse dep label %q in %v", depStr, noSuchTargetRe)
				continue
			}
			from, err := label.Parse(fromStr)
			if err != nil {
				log.Printf("warning: failed to parse from label %q in %v", fromStr)
				continue
			}
			m := &noSuchTarget{
				Filename: match[1],
				Line:     match[2],
				Dep:      dep,
				From:     from,
			}
			addMigration(m)
			log.Printf("Matched NoSuchTarget: %q", m.ID())
			continue
		}

		if match := buildozerRecommentationRe.FindStringSubmatch(line); match != nil {
			toStr := match[2]
			to, err := label.Parse(toStr)
			if err != nil {
				log.Printf("warning: failed to parse to label %q in %v", toStr)
				continue
			}
			m := &buildozerRecommentation{
				Command: match[1],
				To:      to,
			}
			addMigration(m)
			log.Printf("Matched buildozer recommentation: %q", m.ID())
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}

func stripAnsi(str string) string {
	return ansiRe.ReplaceAllString(str, "")
}
