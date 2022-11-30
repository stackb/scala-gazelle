package jarindex

import (
	"fmt"
	"io/ioutil"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
)

func ReadJarIndexProtoFile(filename string) (*jipb.JarIndex, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read jarindex file %q: %w", filename, err)
	}
	index := jipb.JarIndex{}
	if err := proto.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal jarindex proto: %w", err)
	}
	return &index, nil
}

func WriteJarIndexProtoFile(filename string, index *jipb.JarIndex) error {
	data, err := proto.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarindex proto: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarindex proto: %w", err)
	}
	return nil
}

func WriteJarIndexJSONFile(filename string, index *jipb.JarIndex) error {
	data, err := protojson.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarindex json: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarindex json: %w", err)
	}
	return nil
}

func ReadJarFileProtoFile(filename string) (*jipb.JarFile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read jarfile file %q: %w", filename, err)
	}
	index := jipb.JarFile{}
	if err := proto.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("unmarshal jarfile proto: %w", err)
	}
	return &index, nil
}

func WriteJarFileProtoFile(filename string, index *jipb.JarFile) error {
	data, err := proto.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarfile proto: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarfile proto: %w", err)
	}
	return nil
}

func WriteJarFileJSONFile(filename string, index *jipb.JarFile) error {
	data, err := protojson.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal jarfile json: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write jarfile json: %w", err)
	}
	return nil
}
