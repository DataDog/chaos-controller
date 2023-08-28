//
//Copyright The containerd Authors.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.20.1
// source: github.com/containerd/containerd/api/types/transfer/registry.proto

package transfer

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type AuthType int32

const (
	AuthType_NONE AuthType = 0
	// CREDENTIALS is used to exchange username/password for access token
	// using an oauth or "Docker Registry Token" server
	AuthType_CREDENTIALS AuthType = 1
	// REFRESH is used to exchange secret for access token using an oauth
	// or "Docker Registry Token" server
	AuthType_REFRESH AuthType = 2
	// HEADER is used to set the HTTP Authorization header to secret
	// directly for the registry.
	// Value should be `<auth-scheme> <authorization-parameters>`
	AuthType_HEADER AuthType = 3
)

// Enum value maps for AuthType.
var (
	AuthType_name = map[int32]string{
		0: "NONE",
		1: "CREDENTIALS",
		2: "REFRESH",
		3: "HEADER",
	}
	AuthType_value = map[string]int32{
		"NONE":        0,
		"CREDENTIALS": 1,
		"REFRESH":     2,
		"HEADER":      3,
	}
)

func (x AuthType) Enum() *AuthType {
	p := new(AuthType)
	*p = x
	return p
}

func (x AuthType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AuthType) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_enumTypes[0].Descriptor()
}

func (AuthType) Type() protoreflect.EnumType {
	return &file_github_com_containerd_containerd_api_types_transfer_registry_proto_enumTypes[0]
}

func (x AuthType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AuthType.Descriptor instead.
func (AuthType) EnumDescriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescGZIP(), []int{0}
}

type OCIRegistry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Reference string            `protobuf:"bytes,1,opt,name=reference,proto3" json:"reference,omitempty"`
	Resolver  *RegistryResolver `protobuf:"bytes,2,opt,name=resolver,proto3" json:"resolver,omitempty"`
}

func (x *OCIRegistry) Reset() {
	*x = OCIRegistry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OCIRegistry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OCIRegistry) ProtoMessage() {}

func (x *OCIRegistry) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OCIRegistry.ProtoReflect.Descriptor instead.
func (*OCIRegistry) Descriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescGZIP(), []int{0}
}

func (x *OCIRegistry) GetReference() string {
	if x != nil {
		return x.Reference
	}
	return ""
}

func (x *OCIRegistry) GetResolver() *RegistryResolver {
	if x != nil {
		return x.Resolver
	}
	return nil
}

type RegistryResolver struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// auth_stream is used to refer to a stream which auth callbacks may be
	// made on.
	AuthStream string `protobuf:"bytes,1,opt,name=auth_stream,json=authStream,proto3" json:"auth_stream,omitempty"`
	// Headers
	Headers map[string]string `protobuf:"bytes,2,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *RegistryResolver) Reset() {
	*x = RegistryResolver{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RegistryResolver) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegistryResolver) ProtoMessage() {}

func (x *RegistryResolver) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegistryResolver.ProtoReflect.Descriptor instead.
func (*RegistryResolver) Descriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescGZIP(), []int{1}
}

func (x *RegistryResolver) GetAuthStream() string {
	if x != nil {
		return x.AuthStream
	}
	return ""
}

func (x *RegistryResolver) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

// AuthRequest is sent as a callback on a stream
type AuthRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// host is the registry host
	Host string `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	// reference is the namespace and repository name requested from the registry
	Reference string `protobuf:"bytes,2,opt,name=reference,proto3" json:"reference,omitempty"`
	// wwwauthenticate is the HTTP WWW-Authenticate header values returned from the registry
	Wwwauthenticate []string `protobuf:"bytes,3,rep,name=wwwauthenticate,proto3" json:"wwwauthenticate,omitempty"`
}

func (x *AuthRequest) Reset() {
	*x = AuthRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AuthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthRequest) ProtoMessage() {}

func (x *AuthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthRequest.ProtoReflect.Descriptor instead.
func (*AuthRequest) Descriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescGZIP(), []int{2}
}

func (x *AuthRequest) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *AuthRequest) GetReference() string {
	if x != nil {
		return x.Reference
	}
	return ""
}

func (x *AuthRequest) GetWwwauthenticate() []string {
	if x != nil {
		return x.Wwwauthenticate
	}
	return nil
}

type AuthResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AuthType AuthType               `protobuf:"varint,1,opt,name=authType,proto3,enum=containerd.types.transfer.AuthType" json:"authType,omitempty"`
	Secret   string                 `protobuf:"bytes,2,opt,name=secret,proto3" json:"secret,omitempty"`
	Username string                 `protobuf:"bytes,3,opt,name=username,proto3" json:"username,omitempty"`
	ExpireAt *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=expire_at,json=expireAt,proto3" json:"expire_at,omitempty"` // TODO: Stream error
}

func (x *AuthResponse) Reset() {
	*x = AuthResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AuthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthResponse) ProtoMessage() {}

func (x *AuthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthResponse.ProtoReflect.Descriptor instead.
func (*AuthResponse) Descriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescGZIP(), []int{3}
}

func (x *AuthResponse) GetAuthType() AuthType {
	if x != nil {
		return x.AuthType
	}
	return AuthType_NONE
}

func (x *AuthResponse) GetSecret() string {
	if x != nil {
		return x.Secret
	}
	return ""
}

func (x *AuthResponse) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *AuthResponse) GetExpireAt() *timestamppb.Timestamp {
	if x != nil {
		return x.ExpireAt
	}
	return nil
}

var File_github_com_containerd_containerd_api_types_transfer_registry_proto protoreflect.FileDescriptor

var file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDesc = []byte{
	0x0a, 0x42, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x6e,
	0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65,
	0x72, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x66, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64,
	0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x1a,
	0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x74, 0x0a, 0x0b, 0x4f, 0x43, 0x49, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x12,
	0x1c, 0x0a, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x47, 0x0a,
	0x08, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2b, 0x2e, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2e, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x2e, 0x52, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x72, 0x52, 0x08, 0x72, 0x65,
	0x73, 0x6f, 0x6c, 0x76, 0x65, 0x72, 0x22, 0xc3, 0x01, 0x0a, 0x10, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x72, 0x12, 0x1f, 0x0a, 0x0b, 0x61,
	0x75, 0x74, 0x68, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0a, 0x61, 0x75, 0x74, 0x68, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x52, 0x0a, 0x07,
	0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x38, 0x2e,
	0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73,
	0x2e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x72, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73,
	0x1a, 0x3a, 0x0a, 0x0c, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x69, 0x0a, 0x0b,
	0x41, 0x75, 0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x68,
	0x6f, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x12,
	0x1c, 0x0a, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x28, 0x0a,
	0x0f, 0x77, 0x77, 0x77, 0x61, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0f, 0x77, 0x77, 0x77, 0x61, 0x75, 0x74, 0x68, 0x65,
	0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x22, 0xbc, 0x01, 0x0a, 0x0c, 0x41, 0x75, 0x74, 0x68,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3f, 0x0a, 0x08, 0x61, 0x75, 0x74, 0x68,
	0x54, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x23, 0x2e, 0x63, 0x6f, 0x6e,
	0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x74, 0x72,
	0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x54, 0x79, 0x70, 0x65, 0x52,
	0x08, 0x61, 0x75, 0x74, 0x68, 0x54, 0x79, 0x70, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x65, 0x63,
	0x72, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x63, 0x72, 0x65,
	0x74, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x37, 0x0a,
	0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x08, 0x65, 0x78,
	0x70, 0x69, 0x72, 0x65, 0x41, 0x74, 0x2a, 0x3e, 0x0a, 0x08, 0x41, 0x75, 0x74, 0x68, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b,
	0x43, 0x52, 0x45, 0x44, 0x45, 0x4e, 0x54, 0x49, 0x41, 0x4c, 0x53, 0x10, 0x01, 0x12, 0x0b, 0x0a,
	0x07, 0x52, 0x45, 0x46, 0x52, 0x45, 0x53, 0x48, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x48, 0x45,
	0x41, 0x44, 0x45, 0x52, 0x10, 0x03, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2f,
	0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescOnce sync.Once
	file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescData = file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDesc
)

func file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescGZIP() []byte {
	file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescOnce.Do(func() {
		file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescData)
	})
	return file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDescData
}

var file_github_com_containerd_containerd_api_types_transfer_registry_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_github_com_containerd_containerd_api_types_transfer_registry_proto_goTypes = []interface{}{
	(AuthType)(0),                 // 0: containerd.types.transfer.AuthType
	(*OCIRegistry)(nil),           // 1: containerd.types.transfer.OCIRegistry
	(*RegistryResolver)(nil),      // 2: containerd.types.transfer.RegistryResolver
	(*AuthRequest)(nil),           // 3: containerd.types.transfer.AuthRequest
	(*AuthResponse)(nil),          // 4: containerd.types.transfer.AuthResponse
	nil,                           // 5: containerd.types.transfer.RegistryResolver.HeadersEntry
	(*timestamppb.Timestamp)(nil), // 6: google.protobuf.Timestamp
}
var file_github_com_containerd_containerd_api_types_transfer_registry_proto_depIdxs = []int32{
	2, // 0: containerd.types.transfer.OCIRegistry.resolver:type_name -> containerd.types.transfer.RegistryResolver
	5, // 1: containerd.types.transfer.RegistryResolver.headers:type_name -> containerd.types.transfer.RegistryResolver.HeadersEntry
	0, // 2: containerd.types.transfer.AuthResponse.authType:type_name -> containerd.types.transfer.AuthType
	6, // 3: containerd.types.transfer.AuthResponse.expire_at:type_name -> google.protobuf.Timestamp
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_github_com_containerd_containerd_api_types_transfer_registry_proto_init() }
func file_github_com_containerd_containerd_api_types_transfer_registry_proto_init() {
	if File_github_com_containerd_containerd_api_types_transfer_registry_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OCIRegistry); i {
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
		file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RegistryResolver); i {
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
		file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AuthRequest); i {
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
		file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AuthResponse); i {
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
			RawDescriptor: file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_containerd_containerd_api_types_transfer_registry_proto_goTypes,
		DependencyIndexes: file_github_com_containerd_containerd_api_types_transfer_registry_proto_depIdxs,
		EnumInfos:         file_github_com_containerd_containerd_api_types_transfer_registry_proto_enumTypes,
		MessageInfos:      file_github_com_containerd_containerd_api_types_transfer_registry_proto_msgTypes,
	}.Build()
	File_github_com_containerd_containerd_api_types_transfer_registry_proto = out.File
	file_github_com_containerd_containerd_api_types_transfer_registry_proto_rawDesc = nil
	file_github_com_containerd_containerd_api_types_transfer_registry_proto_goTypes = nil
	file_github_com_containerd_containerd_api_types_transfer_registry_proto_depIdxs = nil
}
