package protobuf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type marshaler func(m protoreflect.ProtoMessage) ([]byte, error)
type unmarshaler func(b []byte, m protoreflect.ProtoMessage) error

func unmarshalerForFilename(filename string) unmarshaler {
	if filepath.Ext(filename) == ".json" {
		return protojson.Unmarshal
	}
	if filepath.Ext(filename) == ".pbtext" {
		return prototext.Unmarshal
	}
	return proto.Unmarshal
}

func marshalerForFilename(filename string) marshaler {
	if filepath.Ext(filename) == ".json" {
		return protojson.Marshal
	}
	if filepath.Ext(filename) == ".pbtext" {
		return prototext.Marshal
	}
	return proto.Marshal
}

func ReadFile(filename string, message protoreflect.ProtoMessage) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read %q: %w", filename, err)
	}
	if err := unmarshalerForFilename(filename)(data, message); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

func WriteFile(filename string, message protoreflect.ProtoMessage) error {
	if filepath.Ext(filename) == ".json" {
		return WritePrettyJSONFile(filename, message)
	}
	data, err := marshalerForFilename(filename)(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func ReadFrom(filename string, message protoreflect.ProtoMessage, in io.Reader) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read %q: %w", filename, err)
	}
	if err := unmarshalerForFilename(filename)(data, message); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

func WriteTo(filename string, message protoreflect.ProtoMessage, out io.Writer) error {
	data, err := marshalerForFilename(filename)(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := out.Write(data); err != nil {
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
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func WriteStableJSONFile(filename string, message protoreflect.ProtoMessage) error {
	data, err := StableJSON(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(filename, []byte(data), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func StableJSON(message protoreflect.ProtoMessage) (string, error) {
	marshaler := protojson.MarshalOptions{
		Multiline:       false,
		Indent:          "",
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	var rm json.RawMessage = data
	data2, err := json.MarshalIndent(rm, "", " ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(data2), nil
}

// WriteDelimitedTo writes a length-delimited protobuf message to a writer
func WriteDelimitedTo(msg proto.Message, w io.Writer) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// Write the size as a varint
	sizeBytes := make([]byte, protowire.SizeVarint(uint64(len(data))))
	protowire.AppendVarint(sizeBytes[:0], uint64(len(data)))

	// Write size followed by message data
	if _, err := w.Write(sizeBytes); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}

// ReadDelimitedFrom reads a length-delimited protobuf message from a reader
func ReadDelimitedFrom(msg proto.Message, r io.Reader) error {
	// Read the size varint
	var sizeBuf [10]byte // Maximum size of a varint
	var n int
	var err error

	// Read first byte to determine varint length
	if n, err = r.Read(sizeBuf[:1]); err != nil {
		if err == io.EOF {
			return io.EOF
		}
		return err
	}

	size, bytesRead := protowire.ConsumeVarint(sizeBuf[:n])
	if bytesRead < 0 {
		// Need more bytes for the varint
		var i int
		for i = 1; i < len(sizeBuf) && bytesRead < 0; i++ {
			if _, err = r.Read(sizeBuf[i : i+1]); err != nil {
				return err
			}
			size, bytesRead = protowire.ConsumeVarint(sizeBuf[:i+1])
		}
		if bytesRead < 0 {
			return io.ErrUnexpectedEOF
		}
	}

	// Read the message data
	data := make([]byte, size)
	if _, err = io.ReadFull(r, data); err != nil {
		return err
	}

	// Unmarshal the message
	return proto.Unmarshal(data, msg)
}
