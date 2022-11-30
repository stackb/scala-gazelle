package scalaparse

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func ReadScalaParseRuleListFile(filename string) (*sppb.RuleList, error) {
	if filepath.Ext(filename) == ".json" {
		return ReadScalaParseRuleListJSONFile(filename)
	} else {
		return ReadScalaParseRuleListProtoFile(filename)
	}
}

func ReadScalaParseRuleListProtoFile(filename string) (*sppb.RuleList, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read ScalaSourceIndex file %q: %w", filename, err)
	}
	index := sppb.RuleList{}
	if err := proto.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal ScalaSourceIndex proto: %w", err)
	}
	return &index, nil
}

func ReadScalaParseRuleListJSONFile(filename string) (*sppb.RuleList, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read ScalaSourceIndex file %q: %w", filename, err)
	}
	index := sppb.RuleList{}
	if err := protojson.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal ScalaSourceIndex json: %w", err)
	}
	return &index, nil
}

func WriteScalaParseRuleListFile(filename string, index *sppb.RuleList) error {
	if filepath.Ext(filename) == ".json" {
		return WriteScalaParseRuleListJSONFile(filename, index)
	} else {
		return WriteScalaParseRuleListProtoFile(filename, index)
	}
}

func WriteScalaParseRuleListProtoFile(filename string, index *sppb.RuleList) error {
	data, err := proto.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal ScalaSourceIndex proto: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write ScalaSourceIndex proto: %w", err)
	}
	return nil
}

func WriteScalaParseRuleListJSONFile(filename string, index *sppb.RuleList) error {
	data, err := protojson.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal ScalaSourceIndex json: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write ScalaSourceIndex json: %w", err)
	}
	return nil
}
