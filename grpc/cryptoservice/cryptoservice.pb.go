// Code generated by protoc-gen-go. DO NOT EDIT.
// source: cryptoservice.proto

package cryptoservice

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
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

type PaddingType int32

const (
	PaddingType_Pkcs PaddingType = 0
	PaddingType_Pss  PaddingType = 1
)

var PaddingType_name = map[int32]string{
	0: "Pkcs",
	1: "Pss",
}
var PaddingType_value = map[string]int32{
	"Pkcs": 0,
	"Pss":  1,
}

func (x PaddingType) String() string {
	return proto.EnumName(PaddingType_name, int32(x))
}
func (PaddingType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_cryptoservice_7444e5daee92e3d0, []int{0}
}

type DataToSign struct {
	RawData              []byte      `protobuf:"bytes,1,opt,name=rawData,proto3" json:"rawData,omitempty"`
	KeyLabel             string      `protobuf:"bytes,2,opt,name=keyLabel,proto3" json:"keyLabel,omitempty"`
	HashAlgorithm        string      `protobuf:"bytes,3,opt,name=hashAlgorithm,proto3" json:"hashAlgorithm,omitempty"`
	Padding              PaddingType `protobuf:"varint,4,opt,name=padding,proto3,enum=CryptoGear.Grpc.PaddingType" json:"padding,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *DataToSign) Reset()         { *m = DataToSign{} }
func (m *DataToSign) String() string { return proto.CompactTextString(m) }
func (*DataToSign) ProtoMessage()    {}
func (*DataToSign) Descriptor() ([]byte, []int) {
	return fileDescriptor_cryptoservice_7444e5daee92e3d0, []int{0}
}
func (m *DataToSign) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DataToSign.Unmarshal(m, b)
}
func (m *DataToSign) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DataToSign.Marshal(b, m, deterministic)
}
func (dst *DataToSign) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DataToSign.Merge(dst, src)
}
func (m *DataToSign) XXX_Size() int {
	return xxx_messageInfo_DataToSign.Size(m)
}
func (m *DataToSign) XXX_DiscardUnknown() {
	xxx_messageInfo_DataToSign.DiscardUnknown(m)
}

var xxx_messageInfo_DataToSign proto.InternalMessageInfo

func (m *DataToSign) GetRawData() []byte {
	if m != nil {
		return m.RawData
	}
	return nil
}

func (m *DataToSign) GetKeyLabel() string {
	if m != nil {
		return m.KeyLabel
	}
	return ""
}

func (m *DataToSign) GetHashAlgorithm() string {
	if m != nil {
		return m.HashAlgorithm
	}
	return ""
}

func (m *DataToSign) GetPadding() PaddingType {
	if m != nil {
		return m.Padding
	}
	return PaddingType_Pkcs
}

type SignedData struct {
	RawData              []byte   `protobuf:"bytes,1,opt,name=rawData,proto3" json:"rawData,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SignedData) Reset()         { *m = SignedData{} }
func (m *SignedData) String() string { return proto.CompactTextString(m) }
func (*SignedData) ProtoMessage()    {}
func (*SignedData) Descriptor() ([]byte, []int) {
	return fileDescriptor_cryptoservice_7444e5daee92e3d0, []int{1}
}
func (m *SignedData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SignedData.Unmarshal(m, b)
}
func (m *SignedData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SignedData.Marshal(b, m, deterministic)
}
func (dst *SignedData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignedData.Merge(dst, src)
}
func (m *SignedData) XXX_Size() int {
	return xxx_messageInfo_SignedData.Size(m)
}
func (m *SignedData) XXX_DiscardUnknown() {
	xxx_messageInfo_SignedData.DiscardUnknown(m)
}

var xxx_messageInfo_SignedData proto.InternalMessageInfo

func (m *SignedData) GetRawData() []byte {
	if m != nil {
		return m.RawData
	}
	return nil
}

func init() {
	proto.RegisterType((*DataToSign)(nil), "CryptoGear.Grpc.DataToSign")
	proto.RegisterType((*SignedData)(nil), "CryptoGear.Grpc.SignedData")
	proto.RegisterEnum("CryptoGear.Grpc.PaddingType", PaddingType_name, PaddingType_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// CryptoGearClient is the client API for CryptoGear service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type CryptoGearClient interface {
	CreateSignature(ctx context.Context, in *DataToSign, opts ...grpc.CallOption) (*SignedData, error)
}

type cryptoGearClient struct {
	cc *grpc.ClientConn
}

func NewCryptoGearClient(cc *grpc.ClientConn) CryptoGearClient {
	return &cryptoGearClient{cc}
}

func (c *cryptoGearClient) CreateSignature(ctx context.Context, in *DataToSign, opts ...grpc.CallOption) (*SignedData, error) {
	out := new(SignedData)
	err := c.cc.Invoke(ctx, "/CryptoGear.Grpc.CryptoGear/createSignature", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CryptoGearServer is the server API for CryptoGear service.
type CryptoGearServer interface {
	CreateSignature(context.Context, *DataToSign) (*SignedData, error)
}

func RegisterCryptoGearServer(s *grpc.Server, srv CryptoGearServer) {
	s.RegisterService(&_CryptoGear_serviceDesc, srv)
}

func _CryptoGear_CreateSignature_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataToSign)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CryptoGearServer).CreateSignature(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/CryptoGear.Grpc.CryptoGear/CreateSignature",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CryptoGearServer).CreateSignature(ctx, req.(*DataToSign))
	}
	return interceptor(ctx, in, info, handler)
}

var _CryptoGear_serviceDesc = grpc.ServiceDesc{
	ServiceName: "CryptoGear.Grpc.CryptoGear",
	HandlerType: (*CryptoGearServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "createSignature",
			Handler:    _CryptoGear_CreateSignature_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cryptoservice.proto",
}

func init() { proto.RegisterFile("cryptoservice.proto", fileDescriptor_cryptoservice_7444e5daee92e3d0) }

var fileDescriptor_cryptoservice_7444e5daee92e3d0 = []byte{
	// 254 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4e, 0x2e, 0xaa, 0x2c,
	0x28, 0xc9, 0x2f, 0x4e, 0x2d, 0x2a, 0xcb, 0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17,
	0xe2, 0x77, 0x06, 0x0b, 0xba, 0xa7, 0x26, 0x16, 0xe9, 0xb9, 0x17, 0x15, 0x24, 0x2b, 0x2d, 0x60,
	0xe4, 0xe2, 0x72, 0x49, 0x2c, 0x49, 0x0c, 0xc9, 0x0f, 0xce, 0x4c, 0xcf, 0x13, 0x92, 0xe0, 0x62,
	0x2f, 0x4a, 0x2c, 0x07, 0x09, 0x48, 0x30, 0x2a, 0x30, 0x6a, 0xf0, 0x04, 0xc1, 0xb8, 0x42, 0x52,
	0x5c, 0x1c, 0xd9, 0xa9, 0x95, 0x3e, 0x89, 0x49, 0xa9, 0x39, 0x12, 0x4c, 0x0a, 0x8c, 0x1a, 0x9c,
	0x41, 0x70, 0xbe, 0x90, 0x0a, 0x17, 0x6f, 0x46, 0x62, 0x71, 0x86, 0x63, 0x4e, 0x7a, 0x7e, 0x51,
	0x66, 0x49, 0x46, 0xae, 0x04, 0x33, 0x58, 0x01, 0xaa, 0xa0, 0x90, 0x19, 0x17, 0x7b, 0x41, 0x62,
	0x4a, 0x4a, 0x66, 0x5e, 0xba, 0x04, 0x8b, 0x02, 0xa3, 0x06, 0x9f, 0x91, 0x8c, 0x1e, 0x9a, 0x6b,
	0xf4, 0x02, 0x20, 0xf2, 0x21, 0x95, 0x05, 0xa9, 0x41, 0x30, 0xc5, 0x4a, 0x6a, 0x5c, 0x5c, 0x20,
	0xb7, 0xa5, 0xa6, 0x80, 0xdd, 0x81, 0xd3, 0x85, 0x5a, 0x0a, 0x5c, 0xdc, 0x48, 0xfa, 0x85, 0x38,
	0xb8, 0x58, 0x02, 0xb2, 0x93, 0x8b, 0x05, 0x18, 0x84, 0xd8, 0xb9, 0x98, 0x03, 0x8a, 0x8b, 0x05,
	0x18, 0x8d, 0x22, 0xb9, 0xb8, 0x10, 0x36, 0x0a, 0x79, 0x73, 0xf1, 0x27, 0x17, 0xa5, 0x26, 0x96,
	0xa4, 0x82, 0x4c, 0x4f, 0x2c, 0x29, 0x2d, 0x4a, 0x15, 0x92, 0xc6, 0x70, 0x11, 0x22, 0x6c, 0xa4,
	0x30, 0x25, 0x11, 0xce, 0x72, 0xe2, 0x8f, 0xe2, 0x45, 0x09, 0xef, 0x24, 0x36, 0x70, 0x80, 0x1b,
	0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x66, 0x98, 0x2b, 0xbf, 0x87, 0x01, 0x00, 0x00,
}