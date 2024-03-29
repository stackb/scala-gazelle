// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.14.0
// source: build/stack/gazelle/scala/parse/compiler.proto

package parse

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Severity int32

const (
	Severity_SEVERITY_UNKNOWN Severity = 0
	Severity_INFO             Severity = 1
	Severity_WARN             Severity = 2
	Severity_ERROR            Severity = 3
)

// Enum value maps for Severity.
var (
	Severity_name = map[int32]string{
		0: "SEVERITY_UNKNOWN",
		1: "INFO",
		2: "WARN",
		3: "ERROR",
	}
	Severity_value = map[string]int32{
		"SEVERITY_UNKNOWN": 0,
		"INFO":             1,
		"WARN":             2,
		"ERROR":            3,
	}
)

func (x Severity) Enum() *Severity {
	p := new(Severity)
	*p = x
	return p
}

func (x Severity) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Severity) Descriptor() protoreflect.EnumDescriptor {
	return file_build_stack_gazelle_scala_parse_compiler_proto_enumTypes[0].Descriptor()
}

func (Severity) Type() protoreflect.EnumType {
	return &file_build_stack_gazelle_scala_parse_compiler_proto_enumTypes[0]
}

func (x Severity) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Severity.Descriptor instead.
func (Severity) EnumDescriptor() ([]byte, []int) {
	return file_build_stack_gazelle_scala_parse_compiler_proto_rawDescGZIP(), []int{0}
}

type CompileRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Dir       string   `protobuf:"bytes,1,opt,name=dir,proto3" json:"dir,omitempty"`
	Filenames []string `protobuf:"bytes,2,rep,name=filenames,proto3" json:"filenames,omitempty"`
}

func (x *CompileRequest) Reset() {
	*x = CompileRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CompileRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CompileRequest) ProtoMessage() {}

func (x *CompileRequest) ProtoReflect() protoreflect.Message {
	mi := &file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CompileRequest.ProtoReflect.Descriptor instead.
func (*CompileRequest) Descriptor() ([]byte, []int) {
	return file_build_stack_gazelle_scala_parse_compiler_proto_rawDescGZIP(), []int{0}
}

func (x *CompileRequest) GetDir() string {
	if x != nil {
		return x.Dir
	}
	return ""
}

func (x *CompileRequest) GetFilenames() []string {
	if x != nil {
		return x.Filenames
	}
	return nil
}

type Diagnostic struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Severity Severity `protobuf:"varint,1,opt,name=severity,proto3,enum=build.stack.gazelle.scala.parse.Severity" json:"severity,omitempty"`
	Source   string   `protobuf:"bytes,2,opt,name=source,proto3" json:"source,omitempty"`
	Line     int32    `protobuf:"varint,3,opt,name=line,proto3" json:"line,omitempty"`
	Message  string   `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *Diagnostic) Reset() {
	*x = Diagnostic{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Diagnostic) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Diagnostic) ProtoMessage() {}

func (x *Diagnostic) ProtoReflect() protoreflect.Message {
	mi := &file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Diagnostic.ProtoReflect.Descriptor instead.
func (*Diagnostic) Descriptor() ([]byte, []int) {
	return file_build_stack_gazelle_scala_parse_compiler_proto_rawDescGZIP(), []int{1}
}

func (x *Diagnostic) GetSeverity() Severity {
	if x != nil {
		return x.Severity
	}
	return Severity_SEVERITY_UNKNOWN
}

func (x *Diagnostic) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

func (x *Diagnostic) GetLine() int32 {
	if x != nil {
		return x.Line
	}
	return 0
}

func (x *Diagnostic) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type CompileResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Diagnostics   []*Diagnostic `protobuf:"bytes,1,rep,name=diagnostics,proto3" json:"diagnostics,omitempty"`
	Error         string        `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	ElapsedMillis int64         `protobuf:"varint,3,opt,name=elapsed_millis,json=elapsedMillis,proto3" json:"elapsed_millis,omitempty"`
}

func (x *CompileResponse) Reset() {
	*x = CompileResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CompileResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CompileResponse) ProtoMessage() {}

func (x *CompileResponse) ProtoReflect() protoreflect.Message {
	mi := &file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CompileResponse.ProtoReflect.Descriptor instead.
func (*CompileResponse) Descriptor() ([]byte, []int) {
	return file_build_stack_gazelle_scala_parse_compiler_proto_rawDescGZIP(), []int{2}
}

func (x *CompileResponse) GetDiagnostics() []*Diagnostic {
	if x != nil {
		return x.Diagnostics
	}
	return nil
}

func (x *CompileResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

func (x *CompileResponse) GetElapsedMillis() int64 {
	if x != nil {
		return x.ElapsedMillis
	}
	return 0
}

var File_build_stack_gazelle_scala_parse_compiler_proto protoreflect.FileDescriptor

var file_build_stack_gazelle_scala_parse_compiler_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2f, 0x67, 0x61,
	0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2f, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2f, 0x70, 0x61, 0x72, 0x73,
	0x65, 0x2f, 0x63, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x1f, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2e, 0x67, 0x61,
	0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2e, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2e, 0x70, 0x61, 0x72, 0x73,
	0x65, 0x1a, 0x2a, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2f, 0x67,
	0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2f, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2f, 0x70, 0x61, 0x72,
	0x73, 0x65, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x40, 0x0a,
	0x0e, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x10, 0x0a, 0x03, 0x64, 0x69, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x64, 0x69,
	0x72, 0x12, 0x1c, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x22,
	0x99, 0x01, 0x0a, 0x0a, 0x44, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73, 0x74, 0x69, 0x63, 0x12, 0x45,
	0x0a, 0x08, 0x73, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x29, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2e, 0x67,
	0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2e, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2e, 0x70, 0x61, 0x72,
	0x73, 0x65, 0x2e, 0x53, 0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x52, 0x08, 0x73, 0x65, 0x76,
	0x65, 0x72, 0x69, 0x74, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x6c, 0x69, 0x6e, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x6c, 0x69, 0x6e,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x9d, 0x01, 0x0a, 0x0f,
	0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x4d, 0x0a, 0x0b, 0x64, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73, 0x74, 0x69, 0x63, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2e, 0x73, 0x74, 0x61,
	0x63, 0x6b, 0x2e, 0x67, 0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2e, 0x73, 0x63, 0x61, 0x6c, 0x61,
	0x2e, 0x70, 0x61, 0x72, 0x73, 0x65, 0x2e, 0x44, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73, 0x74, 0x69,
	0x63, 0x52, 0x0b, 0x64, 0x69, 0x61, 0x67, 0x6e, 0x6f, 0x73, 0x74, 0x69, 0x63, 0x73, 0x12, 0x14,
	0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x12, 0x25, 0x0a, 0x0e, 0x65, 0x6c, 0x61, 0x70, 0x73, 0x65, 0x64, 0x5f,
	0x6d, 0x69, 0x6c, 0x6c, 0x69, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0d, 0x65, 0x6c,
	0x61, 0x70, 0x73, 0x65, 0x64, 0x4d, 0x69, 0x6c, 0x6c, 0x69, 0x73, 0x2a, 0x3f, 0x0a, 0x08, 0x53,
	0x65, 0x76, 0x65, 0x72, 0x69, 0x74, 0x79, 0x12, 0x14, 0x0a, 0x10, 0x53, 0x45, 0x56, 0x45, 0x52,
	0x49, 0x54, 0x59, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x08, 0x0a,
	0x04, 0x49, 0x4e, 0x46, 0x4f, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x57, 0x41, 0x52, 0x4e, 0x10,
	0x02, 0x12, 0x09, 0x0a, 0x05, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x03, 0x32, 0x7a, 0x0a, 0x08,
	0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x12, 0x6e, 0x0a, 0x07, 0x43, 0x6f, 0x6d, 0x70,
	0x69, 0x6c, 0x65, 0x12, 0x2f, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2e, 0x73, 0x74, 0x61, 0x63,
	0x6b, 0x2e, 0x67, 0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2e, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2e,
	0x70, 0x61, 0x72, 0x73, 0x65, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x30, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2e, 0x73, 0x74, 0x61,
	0x63, 0x6b, 0x2e, 0x67, 0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2e, 0x73, 0x63, 0x61, 0x6c, 0x61,
	0x2e, 0x70, 0x61, 0x72, 0x73, 0x65, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x6a, 0x0a, 0x1f, 0x62, 0x75, 0x69, 0x6c,
	0x64, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2e, 0x67, 0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2e,
	0x73, 0x63, 0x61, 0x6c, 0x61, 0x2e, 0x70, 0x61, 0x72, 0x73, 0x65, 0x50, 0x01, 0x5a, 0x45, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x62,
	0x2f, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2d, 0x67, 0x61, 0x7a, 0x65, 0x6c, 0x6c, 0x65, 0x2f, 0x62,
	0x75, 0x69, 0x6c, 0x64, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2f, 0x67, 0x61, 0x7a, 0x65, 0x6c,
	0x6c, 0x65, 0x2f, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x2f, 0x70, 0x61, 0x72, 0x73, 0x65, 0x3b, 0x70,
	0x61, 0x72, 0x73, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_build_stack_gazelle_scala_parse_compiler_proto_rawDescOnce sync.Once
	file_build_stack_gazelle_scala_parse_compiler_proto_rawDescData = file_build_stack_gazelle_scala_parse_compiler_proto_rawDesc
)

func file_build_stack_gazelle_scala_parse_compiler_proto_rawDescGZIP() []byte {
	file_build_stack_gazelle_scala_parse_compiler_proto_rawDescOnce.Do(func() {
		file_build_stack_gazelle_scala_parse_compiler_proto_rawDescData = protoimpl.X.CompressGZIP(file_build_stack_gazelle_scala_parse_compiler_proto_rawDescData)
	})
	return file_build_stack_gazelle_scala_parse_compiler_proto_rawDescData
}

var file_build_stack_gazelle_scala_parse_compiler_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_build_stack_gazelle_scala_parse_compiler_proto_goTypes = []interface{}{
	(Severity)(0),           // 0: build.stack.gazelle.scala.parse.Severity
	(*CompileRequest)(nil),  // 1: build.stack.gazelle.scala.parse.CompileRequest
	(*Diagnostic)(nil),      // 2: build.stack.gazelle.scala.parse.Diagnostic
	(*CompileResponse)(nil), // 3: build.stack.gazelle.scala.parse.CompileResponse
}
var file_build_stack_gazelle_scala_parse_compiler_proto_depIdxs = []int32{
	0, // 0: build.stack.gazelle.scala.parse.Diagnostic.severity:type_name -> build.stack.gazelle.scala.parse.Severity
	2, // 1: build.stack.gazelle.scala.parse.CompileResponse.diagnostics:type_name -> build.stack.gazelle.scala.parse.Diagnostic
	1, // 2: build.stack.gazelle.scala.parse.Compiler.Compile:input_type -> build.stack.gazelle.scala.parse.CompileRequest
	3, // 3: build.stack.gazelle.scala.parse.Compiler.Compile:output_type -> build.stack.gazelle.scala.parse.CompileResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_build_stack_gazelle_scala_parse_compiler_proto_init() }
func file_build_stack_gazelle_scala_parse_compiler_proto_init() {
	if File_build_stack_gazelle_scala_parse_compiler_proto != nil {
		return
	}
	file_build_stack_gazelle_scala_parse_file_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CompileRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Diagnostic); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CompileResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_build_stack_gazelle_scala_parse_compiler_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_build_stack_gazelle_scala_parse_compiler_proto_goTypes,
		DependencyIndexes: file_build_stack_gazelle_scala_parse_compiler_proto_depIdxs,
		EnumInfos:         file_build_stack_gazelle_scala_parse_compiler_proto_enumTypes,
		MessageInfos:      file_build_stack_gazelle_scala_parse_compiler_proto_msgTypes,
	}.Build()
	File_build_stack_gazelle_scala_parse_compiler_proto = out.File
	file_build_stack_gazelle_scala_parse_compiler_proto_rawDesc = nil
	file_build_stack_gazelle_scala_parse_compiler_proto_goTypes = nil
	file_build_stack_gazelle_scala_parse_compiler_proto_depIdxs = nil
}
