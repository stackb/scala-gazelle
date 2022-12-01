package scalacompile

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func ReadScalaCompileSpec(filename string) (*ScalaCompileSpec, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var spec ScalaCompileSpec
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
