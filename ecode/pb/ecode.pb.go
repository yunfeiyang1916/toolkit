// Code generated by protoc-gen-go. DO NOT EDIT.
// source: ecode.proto

package pb

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
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Error struct {
	DmError              *int32   `protobuf:"varint,1,req,name=dm_error,json=dmError" json:"dm_error,omitempty"`
	ErrMsg               *string  `protobuf:"bytes,2,req,name=err_msg,json=errMsg" json:"err_msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Error) Reset()         { *m = Error{} }
func (m *Error) String() string { return proto.CompactTextString(m) }
func (*Error) ProtoMessage()    {}
func (*Error) Descriptor() ([]byte, []int) {
	return fileDescriptor_ecode_87aadf18c0529e70, []int{0}
}
func (m *Error) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Error.Unmarshal(m, b)
}
func (m *Error) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Error.Marshal(b, m, deterministic)
}
func (dst *Error) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Error.Merge(dst, src)
}
func (m *Error) XXX_Size() int {
	return xxx_messageInfo_Error.Size(m)
}
func (m *Error) XXX_DiscardUnknown() {
	xxx_messageInfo_Error.DiscardUnknown(m)
}

var xxx_messageInfo_Error proto.InternalMessageInfo

func (m *Error) GetDmError() int32 {
	if m != nil && m.DmError != nil {
		return *m.DmError
	}
	return 0
}

func (m *Error) GetErrMsg() string {
	if m != nil && m.ErrMsg != nil {
		return *m.ErrMsg
	}
	return ""
}

func init() {
	proto.RegisterType((*Error)(nil), "pb.Error")
}

func init() { proto.RegisterFile("ecode.proto", fileDescriptor_ecode_87aadf18c0529e70) }

var fileDescriptor_ecode_87aadf18c0529e70 = []byte{
	// 96 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4e, 0x4d, 0xce, 0x4f,
	0x49, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48, 0x52, 0xb2, 0xe6, 0x62, 0x75,
	0x2d, 0x2a, 0xca, 0x2f, 0x12, 0x92, 0xe4, 0xe2, 0x48, 0xc9, 0x8d, 0x4f, 0x05, 0xb1, 0x25, 0x18,
	0x15, 0x98, 0x34, 0x58, 0x83, 0xd8, 0x53, 0x72, 0x21, 0x52, 0xe2, 0x5c, 0xec, 0xa9, 0x45, 0x45,
	0xf1, 0xb9, 0xc5, 0xe9, 0x12, 0x4c, 0x0a, 0x4c, 0x1a, 0x9c, 0x41, 0x6c, 0xa9, 0x45, 0x45, 0xbe,
	0xc5, 0xe9, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x78, 0xd6, 0x43, 0xca, 0x4e, 0x00, 0x00, 0x00,
}
