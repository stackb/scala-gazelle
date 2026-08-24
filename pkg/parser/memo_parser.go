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
	used  map[label.Label]bool
}

func NewMemoParser(next Parser) *MemoParser {
	return &MemoParser{
		next:  next,
		rules: make(map[label.Label]*sppb.Rule),
		used:  make(map[label.Label]bool),
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
		if !p.used[from] {
			p.used[from] = true
			// Seeded rules carry no symbols so that rules deleted from the
			// tree cannot pollute the resolution scope; load symbols on first
			// use, in walk order like a fresh parse.
			if err := p.next.LoadScalaRule(from, rule); err != nil {
				return nil, err
			}
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
	p.used[from] = true

	if debugMemoParser {
		log.Printf("rule cache save: %s (%s)", from, sha256)
	}

	return rule, nil
}

// LoadScalaRule loads the given state.
func (p *MemoParser) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	p.rules[from] = rule
	p.used[from] = true
	return p.next.LoadScalaRule(from, rule)
}

// SeedScalaRule primes the memo with a cached rule without loading its
// symbols into scope.  Symbols load on the first ParseScalaRule hit, so
// cached rules that no longer exist in the tree never contribute symbols.
func (p *MemoParser) SeedScalaRule(from label.Label, rule *sppb.Rule) {
	p.rules[from] = rule
}

// ScalaRules returns the rules used this run (freshly parsed or cache-hit)
// sorted by label.  Seeded-but-unused rules are dropped so that deleted
// rules age out of a persistent cache file.
func (p *MemoParser) ScalaRules() []*sppb.Rule {
	rules := make([]*sppb.Rule, 0, len(p.rules))
	for from, rule := range p.rules {
		if p.used[from] {
			rules = append(rules, rule)
		}
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
		rule.ParseTimeMillis = 0 // reset for easier diff
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
