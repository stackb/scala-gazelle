package jarindex

import (
	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
)

func ReadJarIndexFile(filename string) (*jipb.JarIndex, error) {
	message := jipb.JarIndex{}
	if err := protobuf.ReadFile(filename, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

func WriteJarIndexFile(filename string, message *jipb.JarIndex) error {
	return protobuf.WriteFile(filename, message)
}

func ReadJarFileFile(filename string) (*jipb.JarFile, error) {
	message := jipb.JarFile{}
	if err := protobuf.ReadFile(filename, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

func WriteJarFileFile(filename string, message *jipb.JarFile) error {
	return protobuf.WriteFile(filename, message)
}
