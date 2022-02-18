package index

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func ReadIndexSpec(filename string) (*IndexSpec, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var spec IndexSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &spec, nil
}

func ReadJarSpec(filename string) (*JarSpec, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var spec JarSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &spec, nil
}

func ReadScalaRuleSpec(filename string) (*ScalaRuleSpec, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var spec ScalaRuleSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &spec, nil
}

func ReadScalaRuleIndexSpec(filename string) (*ScalaRuleIndexSpec, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var spec ScalaRuleIndexSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &spec, nil
}

func WriteJSONFile(filename string, spec interface{}) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0o644)
}

// ListFiles - convenience debugging function to log the files under a given dir
func ListFiles(dir string) error {
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
