package scalaparse

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

// MemoParserService is a Parser frontend that uses cached state of the files sha256
// values are up-to-date.
type MemoParserService struct {
	next ParserService
	// a mapping from the rule.Label to the rule.
	rules map[label.Label]*sppb.Rule
}

func NewMemoParserService(next ParserService) *MemoParserService {
	return &MemoParserService{
		next:  next,
		rules: make(map[label.Label]*sppb.Rule),
	}
}

// Start implements part of the scalaparse.Service interface.
func (p *MemoParserService) Start() error {
	return p.next.Start()
}

// Stop implements part of the scalaparse.Service interface.
func (p *MemoParserService) Stop() {
	p.next.Stop()
}

// ReadScalaRule implements part of the scalaparse.Reader interface.
func (p *MemoParserService) ReadScalaRule(from label.Label, rule *sppb.Rule) error {
	p.rules[from] = rule
	return p.next.ReadScalaRule(from, rule)
}

// ScalaRules implements part of the scalaparse.Reader interface.
func (p *MemoParserService) ScalaRules() []*sppb.Rule {
	rules := make([]*sppb.Rule, 0, len(p.rules))
	for _, rule := range p.rules {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		a := rules[i]
		b := rules[j]
		return a.Label < b.Label
	})
	return rules
}

// ParseScalaFiles implements scalaparse.Parser
func (p *MemoParserService) ParseScalaFiles(from label.Label, kind, dir string, srcs ...string) ([]*sppb.File, error) {

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

	rule, ok := p.rules[from]
	if !ok {
		rule = &sppb.Rule{
			Label:  from.String(),
			Kind:   kind,
			Sha256: sha256,
		}
		p.rules[from] = rule
	} else if rule.Sha256 == sha256 {
		return rule.Files, nil
	}
	rule.Sha256 = sha256

	files, err := p.next.ParseScalaFiles(from, kind, dir, srcs...)
	if err != nil {
		return nil, err
	}
	rule.Files = files

	return files, nil
}
