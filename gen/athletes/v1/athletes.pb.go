// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        (unknown)
// source: athletes/v1/athletes.proto

package athletesv1

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

type Athlete struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name       string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	CreateTime *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
}

func (x *Athlete) Reset() {
	*x = Athlete{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Athlete) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Athlete) ProtoMessage() {}

func (x *Athlete) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Athlete.ProtoReflect.Descriptor instead.
func (*Athlete) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{0}
}

func (x *Athlete) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Athlete) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Athlete) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

type CreateAthleteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *CreateAthleteRequest) Reset() {
	*x = CreateAthleteRequest{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateAthleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateAthleteRequest) ProtoMessage() {}

func (x *CreateAthleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateAthleteRequest.ProtoReflect.Descriptor instead.
func (*CreateAthleteRequest) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{1}
}

func (x *CreateAthleteRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type CreateAthleteResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Athlete *Athlete `protobuf:"bytes,1,opt,name=athlete,proto3" json:"athlete,omitempty"`
}

func (x *CreateAthleteResponse) Reset() {
	*x = CreateAthleteResponse{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateAthleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateAthleteResponse) ProtoMessage() {}

func (x *CreateAthleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateAthleteResponse.ProtoReflect.Descriptor instead.
func (*CreateAthleteResponse) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{2}
}

func (x *CreateAthleteResponse) GetAthlete() *Athlete {
	if x != nil {
		return x.Athlete
	}
	return nil
}

type Activity struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name       string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	AthleteId  string                 `protobuf:"bytes,3,opt,name=athlete_id,json=athleteId,proto3" json:"athlete_id,omitempty"`
	CreateTime *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
}

func (x *Activity) Reset() {
	*x = Activity{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Activity) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Activity) ProtoMessage() {}

func (x *Activity) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Activity.ProtoReflect.Descriptor instead.
func (*Activity) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{3}
}

func (x *Activity) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Activity) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Activity) GetAthleteId() string {
	if x != nil {
		return x.AthleteId
	}
	return ""
}

func (x *Activity) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

type CreateActivityRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name      string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	AthleteId string `protobuf:"bytes,2,opt,name=athlete_id,json=athleteId,proto3" json:"athlete_id,omitempty"`
}

func (x *CreateActivityRequest) Reset() {
	*x = CreateActivityRequest{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateActivityRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateActivityRequest) ProtoMessage() {}

func (x *CreateActivityRequest) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateActivityRequest.ProtoReflect.Descriptor instead.
func (*CreateActivityRequest) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{4}
}

func (x *CreateActivityRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *CreateActivityRequest) GetAthleteId() string {
	if x != nil {
		return x.AthleteId
	}
	return ""
}

type CreateActivityResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Activity *Activity `protobuf:"bytes,1,opt,name=activity,proto3" json:"activity,omitempty"`
}

func (x *CreateActivityResponse) Reset() {
	*x = CreateActivityResponse{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateActivityResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateActivityResponse) ProtoMessage() {}

func (x *CreateActivityResponse) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateActivityResponse.ProtoReflect.Descriptor instead.
func (*CreateActivityResponse) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{5}
}

func (x *CreateActivityResponse) GetActivity() *Activity {
	if x != nil {
		return x.Activity
	}
	return nil
}

type BloodLactateMeasure struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id           string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	ActivityId   string                 `protobuf:"bytes,2,opt,name=activity_id,json=activityId,proto3" json:"activity_id,omitempty"`
	MmolPerLiter string                 `protobuf:"bytes,3,opt,name=mmol_per_liter,json=mmolPerLiter,proto3" json:"mmol_per_liter,omitempty"`
	HeartRateBpm int32                  `protobuf:"varint,4,opt,name=heart_rate_bpm,json=heartRateBpm,proto3" json:"heart_rate_bpm,omitempty"`
	CreateTime   *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
}

func (x *BloodLactateMeasure) Reset() {
	*x = BloodLactateMeasure{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BloodLactateMeasure) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BloodLactateMeasure) ProtoMessage() {}

func (x *BloodLactateMeasure) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BloodLactateMeasure.ProtoReflect.Descriptor instead.
func (*BloodLactateMeasure) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{6}
}

func (x *BloodLactateMeasure) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *BloodLactateMeasure) GetActivityId() string {
	if x != nil {
		return x.ActivityId
	}
	return ""
}

func (x *BloodLactateMeasure) GetMmolPerLiter() string {
	if x != nil {
		return x.MmolPerLiter
	}
	return ""
}

func (x *BloodLactateMeasure) GetHeartRateBpm() int32 {
	if x != nil {
		return x.HeartRateBpm
	}
	return 0
}

func (x *BloodLactateMeasure) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

type CreateBloodLactateMeasureRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ActivityId   string `protobuf:"bytes,1,opt,name=activity_id,json=activityId,proto3" json:"activity_id,omitempty"`
	MmolPerLiter string `protobuf:"bytes,2,opt,name=mmol_per_liter,json=mmolPerLiter,proto3" json:"mmol_per_liter,omitempty"`
	HeartRateBpm int32  `protobuf:"varint,3,opt,name=heart_rate_bpm,json=heartRateBpm,proto3" json:"heart_rate_bpm,omitempty"`
}

func (x *CreateBloodLactateMeasureRequest) Reset() {
	*x = CreateBloodLactateMeasureRequest{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateBloodLactateMeasureRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBloodLactateMeasureRequest) ProtoMessage() {}

func (x *CreateBloodLactateMeasureRequest) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBloodLactateMeasureRequest.ProtoReflect.Descriptor instead.
func (*CreateBloodLactateMeasureRequest) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{7}
}

func (x *CreateBloodLactateMeasureRequest) GetActivityId() string {
	if x != nil {
		return x.ActivityId
	}
	return ""
}

func (x *CreateBloodLactateMeasureRequest) GetMmolPerLiter() string {
	if x != nil {
		return x.MmolPerLiter
	}
	return ""
}

func (x *CreateBloodLactateMeasureRequest) GetHeartRateBpm() int32 {
	if x != nil {
		return x.HeartRateBpm
	}
	return 0
}

type CreateBloodLactateMeasureResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BloodLactateMeasure *BloodLactateMeasure `protobuf:"bytes,1,opt,name=blood_lactate_measure,json=bloodLactateMeasure,proto3" json:"blood_lactate_measure,omitempty"`
}

func (x *CreateBloodLactateMeasureResponse) Reset() {
	*x = CreateBloodLactateMeasureResponse{}
	mi := &file_athletes_v1_athletes_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateBloodLactateMeasureResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBloodLactateMeasureResponse) ProtoMessage() {}

func (x *CreateBloodLactateMeasureResponse) ProtoReflect() protoreflect.Message {
	mi := &file_athletes_v1_athletes_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBloodLactateMeasureResponse.ProtoReflect.Descriptor instead.
func (*CreateBloodLactateMeasureResponse) Descriptor() ([]byte, []int) {
	return file_athletes_v1_athletes_proto_rawDescGZIP(), []int{8}
}

func (x *CreateBloodLactateMeasureResponse) GetBloodLactateMeasure() *BloodLactateMeasure {
	if x != nil {
		return x.BloodLactateMeasure
	}
	return nil
}

var File_athletes_v1_athletes_proto protoreflect.FileDescriptor

var file_athletes_v1_athletes_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x74,
	0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x61, 0x74,
	0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x6a, 0x0a, 0x07, 0x41, 0x74,
	0x68, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x3b, 0x0a, 0x0b, 0x63, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x22, 0x2a, 0x0a, 0x14, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x41, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x22, 0x47, 0x0a, 0x15, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x74, 0x68, 0x6c,
	0x65, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2e, 0x0a, 0x07, 0x61,
	0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x61,
	0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x74, 0x68, 0x6c, 0x65,
	0x74, 0x65, 0x52, 0x07, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x22, 0x8a, 0x01, 0x0a, 0x08,
	0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a,
	0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x49, 0x64, 0x12, 0x3b, 0x0a, 0x0b, 0x63,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x63, 0x72,
	0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x22, 0x4a, 0x0a, 0x15, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65,
	0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x74, 0x68, 0x6c, 0x65,
	0x74, 0x65, 0x49, 0x64, 0x22, 0x4b, 0x0a, 0x16, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x63,
	0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x31,
	0x0a, 0x08, 0x61, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x15, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x41,
	0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x08, 0x61, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74,
	0x79, 0x22, 0xcf, 0x01, 0x0a, 0x13, 0x42, 0x6c, 0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63, 0x74, 0x61,
	0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x63, 0x74,
	0x69, 0x76, 0x69, 0x74, 0x79, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a,
	0x61, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x49, 0x64, 0x12, 0x24, 0x0a, 0x0e, 0x6d, 0x6d,
	0x6f, 0x6c, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x6c, 0x69, 0x74, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0c, 0x6d, 0x6d, 0x6f, 0x6c, 0x50, 0x65, 0x72, 0x4c, 0x69, 0x74, 0x65, 0x72,
	0x12, 0x24, 0x0a, 0x0e, 0x68, 0x65, 0x61, 0x72, 0x74, 0x5f, 0x72, 0x61, 0x74, 0x65, 0x5f, 0x62,
	0x70, 0x6d, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x68, 0x65, 0x61, 0x72, 0x74, 0x52,
	0x61, 0x74, 0x65, 0x42, 0x70, 0x6d, 0x12, 0x3b, 0x0a, 0x0b, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x54,
	0x69, 0x6d, 0x65, 0x22, 0x8f, 0x01, 0x0a, 0x20, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6c,
	0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x63, 0x74, 0x69,
	0x76, 0x69, 0x74, 0x79, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x61,
	0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x49, 0x64, 0x12, 0x24, 0x0a, 0x0e, 0x6d, 0x6d, 0x6f,
	0x6c, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x6c, 0x69, 0x74, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0c, 0x6d, 0x6d, 0x6f, 0x6c, 0x50, 0x65, 0x72, 0x4c, 0x69, 0x74, 0x65, 0x72, 0x12,
	0x24, 0x0a, 0x0e, 0x68, 0x65, 0x61, 0x72, 0x74, 0x5f, 0x72, 0x61, 0x74, 0x65, 0x5f, 0x62, 0x70,
	0x6d, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x68, 0x65, 0x61, 0x72, 0x74, 0x52, 0x61,
	0x74, 0x65, 0x42, 0x70, 0x6d, 0x22, 0x79, 0x0a, 0x21, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42,
	0x6c, 0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75,
	0x72, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x54, 0x0a, 0x15, 0x62, 0x6c,
	0x6f, 0x6f, 0x64, 0x5f, 0x6c, 0x61, 0x63, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x6d, 0x65, 0x61, 0x73,
	0x75, 0x72, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x61, 0x74, 0x68, 0x6c,
	0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x6c, 0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63,
	0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x52, 0x13, 0x62, 0x6c, 0x6f,
	0x6f, 0x64, 0x4c, 0x61, 0x63, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65,
	0x32, 0xbf, 0x02, 0x0a, 0x0e, 0x41, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x56, 0x0a, 0x0d, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x74, 0x68,
	0x6c, 0x65, 0x74, 0x65, 0x12, 0x21, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e,
	0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x22, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74,
	0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x74, 0x68, 0x6c,
	0x65, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x59, 0x0a, 0x0e, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x12, 0x22, 0x2e,
	0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x23, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x7a, 0x0a, 0x19, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x42, 0x6c, 0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73,
	0x75, 0x72, 0x65, 0x12, 0x2d, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76,
	0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63,
	0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x6f, 0x64, 0x4c, 0x61, 0x63, 0x74,
	0x61, 0x74, 0x65, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x42, 0xa9, 0x01, 0x0a, 0x0f, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x74, 0x68, 0x6c, 0x65,
	0x74, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x42, 0x0d, 0x41, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x3a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x61, 0x74, 0x7a, 0x6b, 0x6f, 0x72, 0x6e, 0x2f, 0x74, 0x72, 0x61,
	0x69, 0x6c, 0x2d, 0x74, 0x6f, 0x6f, 0x6c, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x61, 0x74, 0x68,
	0x6c, 0x65, 0x74, 0x65, 0x73, 0x2f, 0x76, 0x31, 0x3b, 0x61, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65,
	0x73, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x41, 0x58, 0x58, 0xaa, 0x02, 0x0b, 0x41, 0x74, 0x68, 0x6c,
	0x65, 0x74, 0x65, 0x73, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x0b, 0x41, 0x74, 0x68, 0x6c, 0x65, 0x74,
	0x65, 0x73, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x17, 0x41, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73,
	0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea,
	0x02, 0x0c, 0x41, 0x74, 0x68, 0x6c, 0x65, 0x74, 0x65, 0x73, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_athletes_v1_athletes_proto_rawDescOnce sync.Once
	file_athletes_v1_athletes_proto_rawDescData = file_athletes_v1_athletes_proto_rawDesc
)

func file_athletes_v1_athletes_proto_rawDescGZIP() []byte {
	file_athletes_v1_athletes_proto_rawDescOnce.Do(func() {
		file_athletes_v1_athletes_proto_rawDescData = protoimpl.X.CompressGZIP(file_athletes_v1_athletes_proto_rawDescData)
	})
	return file_athletes_v1_athletes_proto_rawDescData
}

var file_athletes_v1_athletes_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_athletes_v1_athletes_proto_goTypes = []any{
	(*Athlete)(nil),                           // 0: athletes.v1.Athlete
	(*CreateAthleteRequest)(nil),              // 1: athletes.v1.CreateAthleteRequest
	(*CreateAthleteResponse)(nil),             // 2: athletes.v1.CreateAthleteResponse
	(*Activity)(nil),                          // 3: athletes.v1.Activity
	(*CreateActivityRequest)(nil),             // 4: athletes.v1.CreateActivityRequest
	(*CreateActivityResponse)(nil),            // 5: athletes.v1.CreateActivityResponse
	(*BloodLactateMeasure)(nil),               // 6: athletes.v1.BloodLactateMeasure
	(*CreateBloodLactateMeasureRequest)(nil),  // 7: athletes.v1.CreateBloodLactateMeasureRequest
	(*CreateBloodLactateMeasureResponse)(nil), // 8: athletes.v1.CreateBloodLactateMeasureResponse
	(*timestamppb.Timestamp)(nil),             // 9: google.protobuf.Timestamp
}
var file_athletes_v1_athletes_proto_depIdxs = []int32{
	9, // 0: athletes.v1.Athlete.create_time:type_name -> google.protobuf.Timestamp
	0, // 1: athletes.v1.CreateAthleteResponse.athlete:type_name -> athletes.v1.Athlete
	9, // 2: athletes.v1.Activity.create_time:type_name -> google.protobuf.Timestamp
	3, // 3: athletes.v1.CreateActivityResponse.activity:type_name -> athletes.v1.Activity
	9, // 4: athletes.v1.BloodLactateMeasure.create_time:type_name -> google.protobuf.Timestamp
	6, // 5: athletes.v1.CreateBloodLactateMeasureResponse.blood_lactate_measure:type_name -> athletes.v1.BloodLactateMeasure
	1, // 6: athletes.v1.AthleteService.CreateAthlete:input_type -> athletes.v1.CreateAthleteRequest
	4, // 7: athletes.v1.AthleteService.CreateActivity:input_type -> athletes.v1.CreateActivityRequest
	7, // 8: athletes.v1.AthleteService.CreateBloodLactateMeasure:input_type -> athletes.v1.CreateBloodLactateMeasureRequest
	2, // 9: athletes.v1.AthleteService.CreateAthlete:output_type -> athletes.v1.CreateAthleteResponse
	5, // 10: athletes.v1.AthleteService.CreateActivity:output_type -> athletes.v1.CreateActivityResponse
	8, // 11: athletes.v1.AthleteService.CreateBloodLactateMeasure:output_type -> athletes.v1.CreateBloodLactateMeasureResponse
	9, // [9:12] is the sub-list for method output_type
	6, // [6:9] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_athletes_v1_athletes_proto_init() }
func file_athletes_v1_athletes_proto_init() {
	if File_athletes_v1_athletes_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_athletes_v1_athletes_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_athletes_v1_athletes_proto_goTypes,
		DependencyIndexes: file_athletes_v1_athletes_proto_depIdxs,
		MessageInfos:      file_athletes_v1_athletes_proto_msgTypes,
	}.Build()
	File_athletes_v1_athletes_proto = out.File
	file_athletes_v1_athletes_proto_rawDesc = nil
	file_athletes_v1_athletes_proto_goTypes = nil
	file_athletes_v1_athletes_proto_depIdxs = nil
}