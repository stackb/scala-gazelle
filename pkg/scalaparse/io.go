package scalaparse

import (
	"log"
	"os"
	"path/filepath"

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

func ListFiles(dir string) error {
	// ListFiles - convenience debugging function to log the files under a given dir
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		if info.Mode()&os.ModeSymlink > 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			log.Printf("%s -> %s", path, link)
			return nil
		}

		log.Println(path)
		return nil
	})
}
