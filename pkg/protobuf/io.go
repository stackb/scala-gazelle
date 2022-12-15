package protobuf

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type marshaler func(m protoreflect.ProtoMessage) ([]byte, error)
type unmarshaler func(b []byte, m protoreflect.ProtoMessage) error

func unmarshalerForFilename(filename string) unmarshaler {
	if filepath.Ext(filename) == ".json" {
		return protojson.Unmarshal
	}
	return proto.Unmarshal
}

func marshalerForFilename(filename string) marshaler {
	if filepath.Ext(filename) == ".json" {
		return protojson.Marshal
	}
	return proto.Marshal
}

func ReadFile(filename string, message protoreflect.ProtoMessage) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read %q: %w", filename, err)
	}
	if err := unmarshalerForFilename(filename)(data, message); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

func WriteFile(filename string, message protoreflect.ProtoMessage) error {
	data, err := marshalerForFilename(filename)(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func WritePrettyJSONFile(filename string, message protoreflect.ProtoMessage) error {
	marshaler := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}
