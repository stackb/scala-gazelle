package mergeindex

import (
	"fmt"
	"io/ioutil"

	"google.golang.org/protobuf/proto"

	jipb "github.com/stackb/scala-gazelle/api/jarindex"
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
