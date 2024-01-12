// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: disruptionlistener.proto

package disruptionlistener

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DisruptionSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Endpoints []*EndpointSpec `protobuf:"bytes,1,rep,name=endpoints,proto3" json:"endpoints,omitempty"`
}

func (x *DisruptionSpec) Reset() {
	*x = DisruptionSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_disruptionlistener_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DisruptionSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DisruptionSpec) ProtoMessage() {}

func (x *DisruptionSpec) ProtoReflect() protoreflect.Message {
	mi := &file_disruptionlistener_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DisruptionSpec.ProtoReflect.Descriptor instead.
func (*DisruptionSpec) Descriptor() ([]byte, []int) {
	return file_disruptionlistener_proto_rawDescGZIP(), []int{0}
}

func (x *DisruptionSpec) GetEndpoints() []*EndpointSpec {
	if x != nil {
		return x.Endpoints
	}
	return nil
}

type EndpointSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TargetEndpoint string            `protobuf:"bytes,1,opt,name=targetEndpoint,proto3" json:"targetEndpoint,omitempty"`
	Alterations    []*AlterationSpec `protobuf:"bytes,2,rep,name=alterations,proto3" json:"alterations,omitempty"`
}

func (x *EndpointSpec) Reset() {
	*x = EndpointSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_disruptionlistener_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EndpointSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EndpointSpec) ProtoMessage() {}

func (x *EndpointSpec) ProtoReflect() protoreflect.Message {
	mi := &file_disruptionlistener_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EndpointSpec.ProtoReflect.Descriptor instead.
func (*EndpointSpec) Descriptor() ([]byte, []int) {
	return file_disruptionlistener_proto_rawDescGZIP(), []int{1}
}

func (x *EndpointSpec) GetTargetEndpoint() string {
	if x != nil {
		return x.TargetEndpoint
	}
	return ""
}

func (x *EndpointSpec) GetAlterations() []*AlterationSpec {
	if x != nil {
		return x.Alterations
	}
	return nil
}

type AlterationSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ErrorToReturn    string `protobuf:"bytes,1,opt,name=errorToReturn,proto3" json:"errorToReturn,omitempty"`
	OverrideToReturn string `protobuf:"bytes,2,opt,name=overrideToReturn,proto3" json:"overrideToReturn,omitempty"`
	QueryPercent     int32  `protobuf:"varint,3,opt,name=queryPercent,proto3" json:"queryPercent,omitempty"`
}

func (x *AlterationSpec) Reset() {
	*x = AlterationSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_disruptionlistener_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AlterationSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AlterationSpec) ProtoMessage() {}

func (x *AlterationSpec) ProtoReflect() protoreflect.Message {
	mi := &file_disruptionlistener_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AlterationSpec.ProtoReflect.Descriptor instead.
func (*AlterationSpec) Descriptor() ([]byte, []int) {
	return file_disruptionlistener_proto_rawDescGZIP(), []int{2}
}

func (x *AlterationSpec) GetErrorToReturn() string {
	if x != nil {
		return x.ErrorToReturn
	}
	return ""
}

func (x *AlterationSpec) GetOverrideToReturn() string {
	if x != nil {
		return x.OverrideToReturn
	}
	return ""
}

func (x *AlterationSpec) GetQueryPercent() int32 {
	if x != nil {
		return x.QueryPercent
	}
	return 0
}

var File_disruptionlistener_proto protoreflect.FileDescriptor

var file_disruptionlistener_proto_rawDesc = []byte{
	0x0a, 0x18, 0x64, 0x69, 0x73, 0x72, 0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x6c, 0x69, 0x73, 0x74,
	0x65, 0x6e, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x64, 0x69, 0x73, 0x72,
	0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x1a, 0x1b,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x50, 0x0a, 0x0e, 0x44,
	0x69, 0x73, 0x72, 0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x12, 0x3e, 0x0a,
	0x09, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x20, 0x2e, 0x64, 0x69, 0x73, 0x72, 0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x6c, 0x69, 0x73,
	0x74, 0x65, 0x6e, 0x65, 0x72, 0x2e, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x53, 0x70,
	0x65, 0x63, 0x52, 0x09, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x22, 0x7c, 0x0a,
	0x0c, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x53, 0x70, 0x65, 0x63, 0x12, 0x26, 0x0a,
	0x0e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x45, 0x6e, 0x64,
	0x70, 0x6f, 0x69, 0x6e, 0x74, 0x12, 0x44, 0x0a, 0x0b, 0x61, 0x6c, 0x74, 0x65, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x64, 0x69, 0x73,
	0x72, 0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x2e,
	0x41, 0x6c, 0x74, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x52, 0x0b,
	0x61, 0x6c, 0x74, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x86, 0x01, 0x0a, 0x0e,
	0x41, 0x6c, 0x74, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x12, 0x24,
	0x0a, 0x0d, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x54, 0x6f, 0x52, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x54, 0x6f, 0x52, 0x65,
	0x74, 0x75, 0x72, 0x6e, 0x12, 0x2a, 0x0a, 0x10, 0x6f, 0x76, 0x65, 0x72, 0x72, 0x69, 0x64, 0x65,
	0x54, 0x6f, 0x52, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10,
	0x6f, 0x76, 0x65, 0x72, 0x72, 0x69, 0x64, 0x65, 0x54, 0x6f, 0x52, 0x65, 0x74, 0x75, 0x72, 0x6e,
	0x12, 0x22, 0x0a, 0x0c, 0x71, 0x75, 0x65, 0x72, 0x79, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x71, 0x75, 0x65, 0x72, 0x79, 0x50, 0x65, 0x72,
	0x63, 0x65, 0x6e, 0x74, 0x32, 0xa3, 0x01, 0x0a, 0x12, 0x44, 0x69, 0x73, 0x72, 0x75, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x12, 0x47, 0x0a, 0x07, 0x44,
	0x69, 0x73, 0x72, 0x75, 0x70, 0x74, 0x12, 0x22, 0x2e, 0x64, 0x69, 0x73, 0x72, 0x75, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x2e, 0x44, 0x69, 0x73, 0x72,
	0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x12, 0x44, 0x0a, 0x10, 0x52, 0x65, 0x73, 0x65, 0x74, 0x44, 0x69, 0x73,
	0x72, 0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x42, 0x16, 0x5a, 0x14, 0x2e, 0x2f,
	0x64, 0x69, 0x73, 0x72, 0x75, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e,
	0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_disruptionlistener_proto_rawDescOnce sync.Once
	file_disruptionlistener_proto_rawDescData = file_disruptionlistener_proto_rawDesc
)

func file_disruptionlistener_proto_rawDescGZIP() []byte {
	file_disruptionlistener_proto_rawDescOnce.Do(func() {
		file_disruptionlistener_proto_rawDescData = protoimpl.X.CompressGZIP(file_disruptionlistener_proto_rawDescData)
	})
	return file_disruptionlistener_proto_rawDescData
}

var file_disruptionlistener_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_disruptionlistener_proto_goTypes = []interface{}{
	(*DisruptionSpec)(nil), // 0: disruptionlistener.DisruptionSpec
	(*EndpointSpec)(nil),   // 1: disruptionlistener.EndpointSpec
	(*AlterationSpec)(nil), // 2: disruptionlistener.AlterationSpec
	(*emptypb.Empty)(nil),  // 3: google.protobuf.Empty
}
var file_disruptionlistener_proto_depIdxs = []int32{
	1, // 0: disruptionlistener.DisruptionSpec.endpoints:type_name -> disruptionlistener.EndpointSpec
	2, // 1: disruptionlistener.EndpointSpec.alterations:type_name -> disruptionlistener.AlterationSpec
	0, // 2: disruptionlistener.DisruptionListener.Disrupt:input_type -> disruptionlistener.DisruptionSpec
	3, // 3: disruptionlistener.DisruptionListener.ResetDisruptions:input_type -> google.protobuf.Empty
	3, // 4: disruptionlistener.DisruptionListener.Disrupt:output_type -> google.protobuf.Empty
	3, // 5: disruptionlistener.DisruptionListener.ResetDisruptions:output_type -> google.protobuf.Empty
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_disruptionlistener_proto_init() }
func file_disruptionlistener_proto_init() {
	if File_disruptionlistener_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_disruptionlistener_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DisruptionSpec); i {
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
		file_disruptionlistener_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EndpointSpec); i {
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
		file_disruptionlistener_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AlterationSpec); i {
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
			RawDescriptor: file_disruptionlistener_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_disruptionlistener_proto_goTypes,
		DependencyIndexes: file_disruptionlistener_proto_depIdxs,
		MessageInfos:      file_disruptionlistener_proto_msgTypes,
	}.Build()
	File_disruptionlistener_proto = out.File
	file_disruptionlistener_proto_rawDesc = nil
	file_disruptionlistener_proto_goTypes = nil
	file_disruptionlistener_proto_depIdxs = nil
}
