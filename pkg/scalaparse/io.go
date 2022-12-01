package scalaparse

import (
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

func ReadRuleListFile(filename string) (*sppb.RuleList, error) {
	message := sppb.RuleList{}
	if err := protobuf.ReadFile(filename, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

func WriteRuleListFile(filename string, message *sppb.RuleList) error {
	return protobuf.WriteFile(filename, message)
}
