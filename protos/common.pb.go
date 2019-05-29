// Code generated by protoc-gen-go. DO NOT EDIT.
// source: common.proto

package protos

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type DHTNameRecord struct {
	Timestamp            string   `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Data                 []byte   `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Signature            []byte   `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
	Delete               bool     `protobuf:"varint,4,opt,name=delete,proto3" json:"delete,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DHTNameRecord) Reset()         { *m = DHTNameRecord{} }
func (m *DHTNameRecord) String() string { return proto.CompactTextString(m) }
func (*DHTNameRecord) ProtoMessage()    {}
func (*DHTNameRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{0}
}

func (m *DHTNameRecord) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DHTNameRecord.Unmarshal(m, b)
}
func (m *DHTNameRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DHTNameRecord.Marshal(b, m, deterministic)
}
func (m *DHTNameRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DHTNameRecord.Merge(m, src)
}
func (m *DHTNameRecord) XXX_Size() int {
	return xxx_messageInfo_DHTNameRecord.Size(m)
}
func (m *DHTNameRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_DHTNameRecord.DiscardUnknown(m)
}

var xxx_messageInfo_DHTNameRecord proto.InternalMessageInfo

func (m *DHTNameRecord) GetTimestamp() string {
	if m != nil {
		return m.Timestamp
	}
	return ""
}

func (m *DHTNameRecord) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *DHTNameRecord) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *DHTNameRecord) GetDelete() bool {
	if m != nil {
		return m.Delete
	}
	return false
}

type BlockMetadata struct {
	Cid                  []byte   `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	Start                int64    `protobuf:"varint,2,opt,name=start,proto3" json:"start,omitempty"`
	Size                 int64    `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	PaddingStart         int64    `protobuf:"varint,4,opt,name=paddingStart,proto3" json:"paddingStart,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BlockMetadata) Reset()         { *m = BlockMetadata{} }
func (m *BlockMetadata) String() string { return proto.CompactTextString(m) }
func (*BlockMetadata) ProtoMessage()    {}
func (*BlockMetadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{1}
}

func (m *BlockMetadata) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BlockMetadata.Unmarshal(m, b)
}
func (m *BlockMetadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BlockMetadata.Marshal(b, m, deterministic)
}
func (m *BlockMetadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BlockMetadata.Merge(m, src)
}
func (m *BlockMetadata) XXX_Size() int {
	return xxx_messageInfo_BlockMetadata.Size(m)
}
func (m *BlockMetadata) XXX_DiscardUnknown() {
	xxx_messageInfo_BlockMetadata.DiscardUnknown(m)
}

var xxx_messageInfo_BlockMetadata proto.InternalMessageInfo

func (m *BlockMetadata) GetCid() []byte {
	if m != nil {
		return m.Cid
	}
	return nil
}

func (m *BlockMetadata) GetStart() int64 {
	if m != nil {
		return m.Start
	}
	return 0
}

func (m *BlockMetadata) GetSize() int64 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *BlockMetadata) GetPaddingStart() int64 {
	if m != nil {
		return m.PaddingStart
	}
	return 0
}

type CrabObject struct {
	Blocks               map[int64]*BlockMetadata `protobuf:"bytes,1,rep,name=blocks,proto3" json:"blocks,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Mtime                string                   `protobuf:"bytes,2,opt,name=mtime,proto3" json:"mtime,omitempty"`
	Size                 int64                    `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	Key                  []byte                   `protobuf:"bytes,4,opt,name=key,proto3" json:"key,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *CrabObject) Reset()         { *m = CrabObject{} }
func (m *CrabObject) String() string { return proto.CompactTextString(m) }
func (*CrabObject) ProtoMessage()    {}
func (*CrabObject) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{2}
}

func (m *CrabObject) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CrabObject.Unmarshal(m, b)
}
func (m *CrabObject) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CrabObject.Marshal(b, m, deterministic)
}
func (m *CrabObject) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CrabObject.Merge(m, src)
}
func (m *CrabObject) XXX_Size() int {
	return xxx_messageInfo_CrabObject.Size(m)
}
func (m *CrabObject) XXX_DiscardUnknown() {
	xxx_messageInfo_CrabObject.DiscardUnknown(m)
}

var xxx_messageInfo_CrabObject proto.InternalMessageInfo

func (m *CrabObject) GetBlocks() map[int64]*BlockMetadata {
	if m != nil {
		return m.Blocks
	}
	return nil
}

func (m *CrabObject) GetMtime() string {
	if m != nil {
		return m.Mtime
	}
	return ""
}

func (m *CrabObject) GetSize() int64 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *CrabObject) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

type BlockStreamRequest struct {
	Cid                  []byte   `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BlockStreamRequest) Reset()         { *m = BlockStreamRequest{} }
func (m *BlockStreamRequest) String() string { return proto.CompactTextString(m) }
func (*BlockStreamRequest) ProtoMessage()    {}
func (*BlockStreamRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{3}
}

func (m *BlockStreamRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BlockStreamRequest.Unmarshal(m, b)
}
func (m *BlockStreamRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BlockStreamRequest.Marshal(b, m, deterministic)
}
func (m *BlockStreamRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BlockStreamRequest.Merge(m, src)
}
func (m *BlockStreamRequest) XXX_Size() int {
	return xxx_messageInfo_BlockStreamRequest.Size(m)
}
func (m *BlockStreamRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BlockStreamRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BlockStreamRequest proto.InternalMessageInfo

func (m *BlockStreamRequest) GetCid() []byte {
	if m != nil {
		return m.Cid
	}
	return nil
}

type Identity struct {
	PrivKey              []byte   `protobuf:"bytes,1,opt,name=privKey,proto3" json:"privKey,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Identity) Reset()         { *m = Identity{} }
func (m *Identity) String() string { return proto.CompactTextString(m) }
func (*Identity) ProtoMessage()    {}
func (*Identity) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{4}
}

func (m *Identity) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Identity.Unmarshal(m, b)
}
func (m *Identity) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Identity.Marshal(b, m, deterministic)
}
func (m *Identity) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Identity.Merge(m, src)
}
func (m *Identity) XXX_Size() int {
	return xxx_messageInfo_Identity.Size(m)
}
func (m *Identity) XXX_DiscardUnknown() {
	xxx_messageInfo_Identity.DiscardUnknown(m)
}

var xxx_messageInfo_Identity proto.InternalMessageInfo

func (m *Identity) GetPrivKey() []byte {
	if m != nil {
		return m.PrivKey
	}
	return nil
}

func init() {
	proto.RegisterType((*DHTNameRecord)(nil), "protos.DHTNameRecord")
	proto.RegisterType((*BlockMetadata)(nil), "protos.BlockMetadata")
	proto.RegisterType((*CrabObject)(nil), "protos.CrabObject")
	proto.RegisterMapType((map[int64]*BlockMetadata)(nil), "protos.CrabObject.BlocksEntry")
	proto.RegisterType((*BlockStreamRequest)(nil), "protos.BlockStreamRequest")
	proto.RegisterType((*Identity)(nil), "protos.Identity")
}

func init() { proto.RegisterFile("common.proto", fileDescriptor_555bd8c177793206) }

var fileDescriptor_555bd8c177793206 = []byte{
	// 347 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0xed, 0x4a, 0xeb, 0x40,
	0x10, 0x65, 0x9b, 0xb6, 0xb7, 0x9d, 0xa6, 0x50, 0x96, 0x7b, 0x2f, 0xe1, 0x72, 0x91, 0x10, 0x44,
	0x02, 0x42, 0x7f, 0x54, 0x10, 0xf1, 0xa7, 0x1f, 0xa0, 0x88, 0x1f, 0x6c, 0x7d, 0x81, 0x4d, 0x32,
	0x94, 0xd8, 0xe6, 0xc3, 0xdd, 0x69, 0xa5, 0x3e, 0xa6, 0x4f, 0x24, 0x3b, 0x69, 0xa9, 0x45, 0x7f,
	0xed, 0x99, 0x99, 0x33, 0x73, 0x0e, 0x87, 0x05, 0x3f, 0xad, 0x8a, 0xa2, 0x2a, 0xc7, 0xb5, 0xa9,
	0xa8, 0x92, 0x5d, 0x7e, 0x6c, 0xf4, 0x06, 0xc3, 0xab, 0x9b, 0xe7, 0x07, 0x5d, 0xa0, 0xc2, 0xb4,
	0x32, 0x99, 0xfc, 0x0f, 0x7d, 0xca, 0x0b, 0xb4, 0xa4, 0x8b, 0x3a, 0x10, 0xa1, 0x88, 0xfb, 0x6a,
	0xd7, 0x90, 0x12, 0xda, 0x99, 0x26, 0x1d, 0xb4, 0x42, 0x11, 0xfb, 0x8a, 0xb1, 0xdb, 0xb0, 0xf9,
	0xac, 0xd4, 0xb4, 0x34, 0x18, 0x78, 0x3c, 0xd8, 0x35, 0xe4, 0x5f, 0xe8, 0x66, 0xb8, 0x40, 0xc2,
	0xa0, 0x1d, 0x8a, 0xb8, 0xa7, 0x36, 0x55, 0x54, 0xc1, 0xf0, 0x62, 0x51, 0xa5, 0xf3, 0x7b, 0x24,
	0xcd, 0x67, 0x46, 0xe0, 0xa5, 0x79, 0xc6, 0x92, 0xbe, 0x72, 0x50, 0xfe, 0x86, 0x8e, 0x25, 0x6d,
	0x88, 0xd5, 0x3c, 0xd5, 0x14, 0xce, 0x82, 0xcd, 0xdf, 0x1b, 0x25, 0x4f, 0x31, 0x96, 0x11, 0xf8,
	0xb5, 0xce, 0xb2, 0xbc, 0x9c, 0x4d, 0x79, 0xa1, 0xcd, 0xb3, 0xbd, 0x5e, 0xf4, 0x21, 0x00, 0x2e,
	0x8d, 0x4e, 0x1e, 0x93, 0x17, 0x4c, 0x49, 0x9e, 0x42, 0x37, 0x71, 0xfa, 0x36, 0x10, 0xa1, 0x17,
	0x0f, 0x26, 0x07, 0x4d, 0x30, 0x76, 0xbc, 0xe3, 0x8c, 0xd9, 0xa0, 0xbd, 0x2e, 0xc9, 0xac, 0xd5,
	0x86, 0xed, 0x4c, 0x15, 0x2e, 0x0f, 0x36, 0xd5, 0x57, 0x4d, 0xf1, 0xa3, 0xa9, 0x11, 0x78, 0x73,
	0x5c, 0xb3, 0x17, 0x5f, 0x39, 0xf8, 0xef, 0x09, 0x06, 0x5f, 0x4e, 0x6e, 0x09, 0x82, 0x77, 0x1c,
	0x94, 0xc7, 0xd0, 0x59, 0xe9, 0xc5, 0xb2, 0x39, 0x3e, 0x98, 0xfc, 0xd9, 0x7a, 0xda, 0x4b, 0x4a,
	0x35, 0x9c, 0xf3, 0xd6, 0x99, 0x88, 0x8e, 0x40, 0xf2, 0x6c, 0x4a, 0x06, 0x75, 0xa1, 0xf0, 0x75,
	0x89, 0x96, 0xbe, 0x47, 0x19, 0x1d, 0x42, 0xef, 0x36, 0xc3, 0x92, 0x72, 0x5a, 0xcb, 0x00, 0x7e,
	0xd5, 0x26, 0x5f, 0xdd, 0x6d, 0xa4, 0x7d, 0xb5, 0x2d, 0x93, 0xe6, 0x53, 0x9c, 0x7c, 0x06, 0x00,
	0x00, 0xff, 0xff, 0xdd, 0x88, 0x8d, 0x99, 0x2b, 0x02, 0x00, 0x00,
}
