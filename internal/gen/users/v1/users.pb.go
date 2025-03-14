// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        (unknown)
// source: users/v1/users.proto

package usersv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type User struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Id         string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	xxx_hidden_Email      string                 `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	xxx_hidden_Name       string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	xxx_hidden_GivenName  string                 `protobuf:"bytes,4,opt,name=given_name,json=givenName,proto3" json:"given_name,omitempty"`
	xxx_hidden_FamilyName string                 `protobuf:"bytes,5,opt,name=family_name,json=familyName,proto3" json:"family_name,omitempty"`
	xxx_hidden_AvatarUrl  string                 `protobuf:"bytes,6,opt,name=avatar_url,json=avatarUrl,proto3" json:"avatar_url,omitempty"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *User) Reset() {
	*x = User{}
	mi := &file_users_v1_users_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_users_v1_users_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *User) GetId() string {
	if x != nil {
		return x.xxx_hidden_Id
	}
	return ""
}

func (x *User) GetEmail() string {
	if x != nil {
		return x.xxx_hidden_Email
	}
	return ""
}

func (x *User) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *User) GetGivenName() string {
	if x != nil {
		return x.xxx_hidden_GivenName
	}
	return ""
}

func (x *User) GetFamilyName() string {
	if x != nil {
		return x.xxx_hidden_FamilyName
	}
	return ""
}

func (x *User) GetAvatarUrl() string {
	if x != nil {
		return x.xxx_hidden_AvatarUrl
	}
	return ""
}

func (x *User) SetId(v string) {
	x.xxx_hidden_Id = v
}

func (x *User) SetEmail(v string) {
	x.xxx_hidden_Email = v
}

func (x *User) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *User) SetGivenName(v string) {
	x.xxx_hidden_GivenName = v
}

func (x *User) SetFamilyName(v string) {
	x.xxx_hidden_FamilyName = v
}

func (x *User) SetAvatarUrl(v string) {
	x.xxx_hidden_AvatarUrl = v
}

type User_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id         string
	Email      string
	Name       string
	GivenName  string
	FamilyName string
	AvatarUrl  string
}

func (b0 User_builder) Build() *User {
	m0 := &User{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Id = b.Id
	x.xxx_hidden_Email = b.Email
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_GivenName = b.GivenName
	x.xxx_hidden_FamilyName = b.FamilyName
	x.xxx_hidden_AvatarUrl = b.AvatarUrl
	return m0
}

type GetCurrentUserRequest struct {
	state         protoimpl.MessageState `protogen:"opaque.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetCurrentUserRequest) Reset() {
	*x = GetCurrentUserRequest{}
	mi := &file_users_v1_users_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCurrentUserRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCurrentUserRequest) ProtoMessage() {}

func (x *GetCurrentUserRequest) ProtoReflect() protoreflect.Message {
	mi := &file_users_v1_users_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type GetCurrentUserRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

}

func (b0 GetCurrentUserRequest_builder) Build() *GetCurrentUserRequest {
	m0 := &GetCurrentUserRequest{}
	b, x := &b0, m0
	_, _ = b, x
	return m0
}

type GetCurrentUserResponse struct {
	state           protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_User *User                  `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *GetCurrentUserResponse) Reset() {
	*x = GetCurrentUserResponse{}
	mi := &file_users_v1_users_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCurrentUserResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCurrentUserResponse) ProtoMessage() {}

func (x *GetCurrentUserResponse) ProtoReflect() protoreflect.Message {
	mi := &file_users_v1_users_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetCurrentUserResponse) GetUser() *User {
	if x != nil {
		return x.xxx_hidden_User
	}
	return nil
}

func (x *GetCurrentUserResponse) SetUser(v *User) {
	x.xxx_hidden_User = v
}

func (x *GetCurrentUserResponse) HasUser() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_User != nil
}

func (x *GetCurrentUserResponse) ClearUser() {
	x.xxx_hidden_User = nil
}

type GetCurrentUserResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	User *User
}

func (b0 GetCurrentUserResponse_builder) Build() *GetCurrentUserResponse {
	m0 := &GetCurrentUserResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_User = b.User
	return m0
}

var File_users_v1_users_proto protoreflect.FileDescriptor

var file_users_v1_users_proto_rawDesc = []byte{
	0x0a, 0x14, 0x75, 0x73, 0x65, 0x72, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x75, 0x73, 0x65, 0x72, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x75, 0x73, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31,
	0x22, 0x9f, 0x01, 0x0a, 0x04, 0x55, 0x73, 0x65, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61,
	0x69, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x67, 0x69, 0x76, 0x65, 0x6e, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x67, 0x69, 0x76, 0x65, 0x6e, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x66, 0x61, 0x6d, 0x69, 0x6c, 0x79, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x66, 0x61, 0x6d, 0x69, 0x6c, 0x79, 0x4e,
	0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x76, 0x61, 0x74, 0x61, 0x72, 0x5f, 0x75, 0x72,
	0x6c, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x76, 0x61, 0x74, 0x61, 0x72, 0x55,
	0x72, 0x6c, 0x22, 0x17, 0x0a, 0x15, 0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x3c, 0x0a, 0x16, 0x47,
	0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x22, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x55,
	0x73, 0x65, 0x72, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x32, 0x62, 0x0a, 0x0b, 0x55, 0x73, 0x65,
	0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x53, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x43,
	0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x55, 0x73, 0x65, 0x72, 0x12, 0x1f, 0x2e, 0x75, 0x73, 0x65,
	0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x20, 0x2e, 0x75, 0x73,
	0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e,
	0x74, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x9a, 0x01,
	0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x42, 0x0a,
	0x55, 0x73, 0x65, 0x72, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x3d, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x61, 0x74, 0x7a, 0x6b, 0x6f, 0x72,
	0x6e, 0x2f, 0x74, 0x72, 0x61, 0x69, 0x6c, 0x2d, 0x74, 0x6f, 0x6f, 0x6c, 0x73, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x75, 0x73, 0x65, 0x72, 0x73,
	0x2f, 0x76, 0x31, 0x3b, 0x75, 0x73, 0x65, 0x72, 0x73, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x55, 0x58,
	0x58, 0xaa, 0x02, 0x08, 0x55, 0x73, 0x65, 0x72, 0x73, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x08, 0x55,
	0x73, 0x65, 0x72, 0x73, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x14, 0x55, 0x73, 0x65, 0x72, 0x73, 0x5c,
	0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02,
	0x09, 0x55, 0x73, 0x65, 0x72, 0x73, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var file_users_v1_users_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_users_v1_users_proto_goTypes = []any{
	(*User)(nil),                   // 0: users.v1.User
	(*GetCurrentUserRequest)(nil),  // 1: users.v1.GetCurrentUserRequest
	(*GetCurrentUserResponse)(nil), // 2: users.v1.GetCurrentUserResponse
}
var file_users_v1_users_proto_depIdxs = []int32{
	0, // 0: users.v1.GetCurrentUserResponse.user:type_name -> users.v1.User
	1, // 1: users.v1.UserService.GetCurrentUser:input_type -> users.v1.GetCurrentUserRequest
	2, // 2: users.v1.UserService.GetCurrentUser:output_type -> users.v1.GetCurrentUserResponse
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_users_v1_users_proto_init() }
func file_users_v1_users_proto_init() {
	if File_users_v1_users_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_users_v1_users_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_users_v1_users_proto_goTypes,
		DependencyIndexes: file_users_v1_users_proto_depIdxs,
		MessageInfos:      file_users_v1_users_proto_msgTypes,
	}.Build()
	File_users_v1_users_proto = out.File
	file_users_v1_users_proto_rawDesc = nil
	file_users_v1_users_proto_goTypes = nil
	file_users_v1_users_proto_depIdxs = nil
}
