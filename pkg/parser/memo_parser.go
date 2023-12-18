package parser

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

const debugMemoParser = false

// MemoParser is a Parser frontend that uses cached state of the files sha256
// values are up-to-date.
type MemoParser struct {
	next  Parser
	rules map[label.Label]*sppb.Rule
}

func NewMemoParser(next Parser) *MemoParser {
	return &MemoParser{
		next:  next,
		rules: make(map[label.Label]*sppb.Rule),
	}
}

// ParseScalaRule implements parser.Parser
func (p *MemoParser) ParseScalaRule(kind string, from label.Label, dir string, srcs ...string) (*sppb.Rule, error) {
	sort.Strings(srcs)

	var hash bytes.Buffer
	for _, src := range srcs {
		filename := filepath.Join(dir, src)
		sha256, err := collections.FileSha256(filename)
		if err != nil {
			return nil, fmt.Errorf("hashing %s: %w", filename, err)
		}
		if _, err := hash.WriteString(sha256); err != nil {
			return nil, err
		}
	}

	sha256, err := collections.Sha256(&hash)
	if err != nil {
		return nil, fmt.Errorf("computing rule files sha256: %w", err)
	}

	if rule, ok := p.rules[from]; ok && rule.Sha256 == sha256 {
		if debugMemoParser {
			log.Printf("rule cache hit: %s", from)
		}
		return rule, nil
	}
	if debugMemoParser {
		log.Printf("rule cache miss: %s (%s)", from, sha256)
	}
	if len(srcs) == 0 {
		log.Panicf(`while parsing %s %s: no files to parse! (this is a bug)`, kind, from)
	}

	rule, err := p.next.ParseScalaRule(kind, from, dir, srcs...)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		log.Panicf(`while parsing %s %s: ParseScalaRule did not return an error, but the returned rule was nil! (this is a bug) [%v]`, kind, from, srcs)
	}
	rule.Sha256 = sha256
	p.rules[from] = rule

	if debugMemoParser {
		log.Printf("rule cache save: %s (%s)", from, sha256)
	}

	return rule, nil
}

// LoadScalaRule loads the given state.
func (p *MemoParser) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	p.rules[from] = rule
	return p.next.LoadScalaRule(from, rule)
}

// ScalaRules returns a list of all scala rules sorted by label
func (p *MemoParser) ScalaRules() []*sppb.Rule {
	rules := make([]*sppb.Rule, 0, len(p.rules))
	for _, rule := range p.rules {
		rules = append(rules, rule)
	}
	SortRules(rules)
	return rules
}

func SortRules(rules []*sppb.Rule) {
	sort.Slice(rules, func(i, j int) bool {
		a := rules[i]
		b := rules[j]
		return a.Label < b.Label
	})
	for _, rule := range rules {
		sortRuleFiles(rule.Files)
	}
}

func sortRuleFiles(files []*sppb.File) {
	sort.Slice(files, func(i, j int) bool {
		a := files[i]
		b := files[j]
		return a.Filename < b.Filename
	})
	for _, file := range files {
		sort.Strings(file.Imports)
		sort.Strings(file.Packages)
		sort.Strings(file.Classes)
		sort.Strings(file.Objects)
		sort.Strings(file.Traits)
		sort.Strings(file.Types)
		sort.Strings(file.Vals)
		sort.Strings(file.Names)
	}
}
