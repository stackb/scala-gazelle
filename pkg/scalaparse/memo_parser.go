package scalaparse

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/label"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

type lookupScalaRule func(label.Label) (*sppb.Rule, bool)

// MemoParser is a Parser frontend that uses cached state of the files sha256
// values are up-to-date.
type MemoParser struct {
	next         Parser
	ruleRegistry lookupScalaRule
}

func NewMemoParser(ruleRegistry lookupScalaRule, next Parser) *MemoParser {
	return &MemoParser{
		next:         next,
		ruleRegistry: ruleRegistry,
	}
}

// ParseScalaFiles implements scalaparse.Parser
func (p *MemoParser) ParseScalaFiles(from label.Label, kind, dir string, srcs ...string) ([]*sppb.File, error) {

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

	rule, ok := p.ruleRegistry(from)
	if !ok {
		log.Panicf("unknown scala rule: %v", from)
	}

	if rule.Sha256 == sha256 {
		return rule.Files, nil
	}
	rule.Sha256 = sha256

	return p.next.ParseScalaFiles(from, kind, dir, srcs...)
}
