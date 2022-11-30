package sourceindex

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	sipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/sourceindex"
)

func ReadScalaSourceIndexFile(filename string) (*sipb.ScalaIndex, error) {
	if filepath.Ext(filename) == ".json" {
		return ReadScalaSourceIndexJSONFile(filename)
	} else {
		return ReadScalaSourceIndexProtoFile(filename)
	}
}

func ReadScalaSourceIndexProtoFile(filename string) (*sipb.ScalaIndex, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read ScalaSourceIndex file %q: %w", filename, err)
	}
	index := sipb.ScalaIndex{}
	if err := proto.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal ScalaSourceIndex proto: %w", err)
	}
	return &index, nil
}

func ReadScalaSourceIndexJSONFile(filename string) (*sipb.ScalaIndex, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read ScalaSourceIndex file %q: %w", filename, err)
	}
	index := sipb.ScalaIndex{}
	if err := protojson.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal ScalaSourceIndex json: %w", err)
	}
	return &index, nil
}

func WriteScalaSourceIndexFile(filename string, index *sipb.ScalaIndex) error {
	if filepath.Ext(filename) == ".json" {
		return WriteScalaSourceIndexJSONFile(filename, index)
	} else {
		return WriteScalaSourceIndexProtoFile(filename, index)
	}
}

func WriteScalaSourceIndexProtoFile(filename string, index *sipb.ScalaIndex) error {
	data, err := proto.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal ScalaSourceIndex proto: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write ScalaSourceIndex proto: %w", err)
	}
	return nil
}

func WriteScalaSourceIndexJSONFile(filename string, index *sipb.ScalaIndex) error {
	data, err := protojson.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal ScalaSourceIndex json: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write ScalaSourceIndex json: %w", err)
	}
	return nil
}
