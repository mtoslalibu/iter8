// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: google/type/money.proto

package google_type

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Represents an amount of money with its currency type.
type Money struct {
	// The 3-letter currency code defined in ISO 4217.
	CurrencyCode string `protobuf:"bytes,1,opt,name=currency_code,json=currencyCode,proto3" json:"currency_code,omitempty"`
	// The whole units of the amount.
	// For example if `currencyCode` is `"USD"`, then 1 unit is one US dollar.
	Units int64 `protobuf:"varint,2,opt,name=units,proto3" json:"units,omitempty"`
	// Number of nano (10^-9) units of the amount.
	// The value must be between -999,999,999 and +999,999,999 inclusive.
	// If `units` is positive, `nanos` must be positive or zero.
	// If `units` is zero, `nanos` can be positive, zero, or negative.
	// If `units` is negative, `nanos` must be negative or zero.
	// For example $-1.75 is represented as `units`=-1 and `nanos`=-750,000,000.
	Nanos int32 `protobuf:"varint,3,opt,name=nanos,proto3" json:"nanos,omitempty"`
}

func (m *Money) Reset()      { *m = Money{} }
func (*Money) ProtoMessage() {}
func (*Money) Descriptor() ([]byte, []int) {
	return fileDescriptor_f093d3d05ab4bbee, []int{0}
}
func (m *Money) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Money) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Money.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Money) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Money.Merge(m, src)
}
func (m *Money) XXX_Size() int {
	return m.Size()
}
func (m *Money) XXX_DiscardUnknown() {
	xxx_messageInfo_Money.DiscardUnknown(m)
}

var xxx_messageInfo_Money proto.InternalMessageInfo

func (m *Money) GetCurrencyCode() string {
	if m != nil {
		return m.CurrencyCode
	}
	return ""
}

func (m *Money) GetUnits() int64 {
	if m != nil {
		return m.Units
	}
	return 0
}

func (m *Money) GetNanos() int32 {
	if m != nil {
		return m.Nanos
	}
	return 0
}

func init() {
	proto.RegisterType((*Money)(nil), "google.type.Money")
}

func init() { proto.RegisterFile("google/type/money.proto", fileDescriptor_f093d3d05ab4bbee) }

var fileDescriptor_f093d3d05ab4bbee = []byte{
	// 216 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4f, 0xcf, 0xcf, 0x4f,
	0xcf, 0x49, 0xd5, 0x2f, 0xa9, 0x2c, 0x48, 0xd5, 0xcf, 0xcd, 0xcf, 0x4b, 0xad, 0xd4, 0x2b, 0x28,
	0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x86, 0x48, 0xe8, 0x81, 0x24, 0x94, 0x22, 0xb8, 0x58, 0x7d, 0x41,
	0x72, 0x42, 0xca, 0x5c, 0xbc, 0xc9, 0xa5, 0x45, 0x45, 0xa9, 0x79, 0xc9, 0x95, 0xf1, 0xc9, 0xf9,
	0x29, 0xa9, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x3c, 0x30, 0x41, 0xe7, 0xfc, 0x94, 0x54,
	0x21, 0x11, 0x2e, 0xd6, 0xd2, 0xbc, 0xcc, 0x92, 0x62, 0x09, 0x26, 0x05, 0x46, 0x0d, 0xe6, 0x20,
	0x08, 0x07, 0x24, 0x9a, 0x97, 0x98, 0x97, 0x5f, 0x2c, 0xc1, 0xac, 0xc0, 0xa8, 0xc1, 0x1a, 0x04,
	0xe1, 0x38, 0xc5, 0x5c, 0x78, 0x28, 0xc7, 0x70, 0xe3, 0xa1, 0x1c, 0xc3, 0x87, 0x87, 0x72, 0x8c,
	0x0d, 0x8f, 0xe4, 0x18, 0x57, 0x3c, 0x92, 0x63, 0x3c, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39,
	0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x5f, 0x3c, 0x92, 0x63, 0xf8, 0xf0, 0x48, 0x8e, 0x71, 0xc2, 0x63,
	0x39, 0x86, 0x0b, 0x8f, 0xe5, 0x18, 0x6e, 0x3c, 0x96, 0x63, 0xe0, 0xe2, 0x4f, 0xce, 0xcf, 0xd5,
	0x43, 0x72, 0xa0, 0x13, 0x17, 0xd8, 0x79, 0x01, 0x20, 0x97, 0x07, 0x30, 0xfe, 0x60, 0x64, 0x5c,
	0xc4, 0xc4, 0xec, 0x1e, 0x12, 0x90, 0xc4, 0x06, 0xf6, 0x8b, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff,
	0x4e, 0x75, 0x23, 0x69, 0xe6, 0x00, 0x00, 0x00,
}

func (this *Money) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Money)
	if !ok {
		that2, ok := that.(Money)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.CurrencyCode != that1.CurrencyCode {
		return false
	}
	if this.Units != that1.Units {
		return false
	}
	if this.Nanos != that1.Nanos {
		return false
	}
	return true
}
func (this *Money) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&google_type.Money{")
	s = append(s, "CurrencyCode: "+fmt.Sprintf("%#v", this.CurrencyCode)+",\n")
	s = append(s, "Units: "+fmt.Sprintf("%#v", this.Units)+",\n")
	s = append(s, "Nanos: "+fmt.Sprintf("%#v", this.Nanos)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringMoney(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Money) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Money) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Money) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Nanos != 0 {
		i = encodeVarintMoney(dAtA, i, uint64(m.Nanos))
		i--
		dAtA[i] = 0x18
	}
	if m.Units != 0 {
		i = encodeVarintMoney(dAtA, i, uint64(m.Units))
		i--
		dAtA[i] = 0x10
	}
	if len(m.CurrencyCode) > 0 {
		i -= len(m.CurrencyCode)
		copy(dAtA[i:], m.CurrencyCode)
		i = encodeVarintMoney(dAtA, i, uint64(len(m.CurrencyCode)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMoney(dAtA []byte, offset int, v uint64) int {
	offset -= sovMoney(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Money) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.CurrencyCode)
	if l > 0 {
		n += 1 + l + sovMoney(uint64(l))
	}
	if m.Units != 0 {
		n += 1 + sovMoney(uint64(m.Units))
	}
	if m.Nanos != 0 {
		n += 1 + sovMoney(uint64(m.Nanos))
	}
	return n
}

func sovMoney(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMoney(x uint64) (n int) {
	return sovMoney(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Money) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Money{`,
		`CurrencyCode:` + fmt.Sprintf("%v", this.CurrencyCode) + `,`,
		`Units:` + fmt.Sprintf("%v", this.Units) + `,`,
		`Nanos:` + fmt.Sprintf("%v", this.Nanos) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringMoney(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Money) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMoney
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Money: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Money: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CurrencyCode", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMoney
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthMoney
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMoney
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CurrencyCode = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Units", wireType)
			}
			m.Units = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMoney
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Units |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nanos", wireType)
			}
			m.Nanos = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMoney
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nanos |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMoney(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMoney
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthMoney
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipMoney(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMoney
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowMoney
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowMoney
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthMoney
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthMoney
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowMoney
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipMoney(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthMoney
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthMoney = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMoney   = fmt.Errorf("proto: integer overflow")
)