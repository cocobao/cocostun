package stun

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	//属性头长度固定值4字节
	attributeHeaderSize = 4

	//消息头长度固定值20字节
	messageHeaderSize = 20

	//TransactionID长度值12字节
	TransactionIDSize = 12 // 96 bit
)

var (
	ErrUnexpectedHeaderEOF = errors.New("unexpected EOF: not enough bytes to read header")
	ErrAttributeNotFound   = errors.New("attribute not found")

	bin = binary.BigEndian
)

//Message 协议格式
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |0 0|     STUN Message Type     |         Message Length        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Magic Cookie                          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// |                     Transaction ID (96 bits)                  |
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type Message struct {
	//消息类型
	Type MessageType
	//消息长度
	Length uint32 // len(Raw) not including header
	//TransactionID 类似于消息id
	TransactionID [TransactionIDSize]byte
	//属性
	Attributes Attributes
	Raw        []byte
}

//根据属性类型获取属性Value值
func (m *Message) Get(t AttrType) ([]byte, error) {
	v, ok := m.Attributes.Get(t)
	if !ok {
		return nil, ErrAttributeNotFound
	}
	return v.Value, nil
}

//生成随机消息id值
func (m *Message) NewTransactionID() error {
	_, err := io.ReadFull(rand.Reader, m.TransactionID[:])
	if err == nil {
		m.WriteTransactionID()
	}
	return err
}

//写入TransactionID到buff
func (m *Message) WriteTransactionID() {
	copy(m.Raw[8:messageHeaderSize], m.TransactionID[:])
}

//设置类型值
func (m *Message) SetType(t MessageType) {
	m.Type = t
	m.WriteType()
}

//写入类型值到buff
func (m *Message) WriteType() {
	bin.PutUint16(m.Raw[0:2], m.Type.Value()) // message type
}

//读取的数据根据协议解析
func (m *Message) Decode() error {
	buf := m.Raw

	//消息长度不应该小于协议头长度
	if len(buf) < messageHeaderSize {
		return ErrUnexpectedHeaderEOF
	}
	var (
		//前两字节,读出头部两字节,包含顶部00以及messageType
		t = bin.Uint16(buf[0:2])
		//第二个两字节，包含消息长度,不包括stun头长度
		size = int(bin.Uint16(buf[2:4]))
		//cookie 固定值0x2112A442
		cookie = bin.Uint32(buf[4:8]) // last 4 bytes
		//完整包长度
		fullSize = messageHeaderSize + size // len(m.Raw)
	)

	//cookie 固定值0x2112A442
	if cookie != magicCookie {
		return fmt.Errorf("%x is invalid magic cookie (should be %x)", cookie, magicCookie)
	}

	//buf数据长度不应小于计算整包长度
	if len(buf) < fullSize {
		return fmt.Errorf("buffer length %d is less than %d (expected message size)", len(buf), fullSize)
	}

	//解析出消息类型字段
	m.Type.ReadValue(t)
	//消息长度
	m.Length = uint32(size)
	//拷贝出16字节TransactionID
	copy(m.TransactionID[:], buf[8:messageHeaderSize])

	m.Attributes = m.Attributes[:0]
	var (
		offset = 0
		b      = buf[messageHeaderSize:fullSize]
	)
	//解析所有属性值
	for offset < size {
		//剩下的数据长度值判断
		if len(b) < attributeHeaderSize {
			return fmt.Errorf("buffer length %d is less than %d (expected header size)", len(b), attributeHeaderSize)
		}
		var (
			a = RawAttribute{
				//读取前两字节为属性类型值
				Type: AttrType(bin.Uint16(b[0:2])),
				//读取属性数据长度值
				Length: bin.Uint16(b[2:4]),
			}
			//属性数据长度
			aL = int(a.Length)
			//属性的value必须是4字节对齐，不够用padding补充
			aBuffL = nearestPaddedValueLength(aL) // expected buffer length (with padding)
		)
		b = b[attributeHeaderSize:]
		//数据便宜值跳过头部字节数
		offset += attributeHeaderSize
		//检测属性数据buf长度是否合法
		if len(b) < aBuffL {
			return fmt.Errorf("buffer length %d is less than %d (expected value size for %s)", len(b), aBuffL, a.Type)
		}

		//读取属性值
		a.Value = b[:aL]

		//数据便宜值应该是加上属性字节对齐后长度。而不是value的实际长度
		offset += aBuffL
		b = b[aBuffL:]

		m.Attributes = append(m.Attributes, a)
	}
	return nil
}

//写入属性
func (m *Message) WriteAttributes() {
	for _, a := range m.Attributes {
		m.Add(a.Type, a.Value)
	}
}

func (m *Message) Add(t AttrType, v []byte) {
	// Allocating buffer for TLV (type-length-value).
	// T = t, L = len(v), V = v.
	// m.Raw will look like:
	// [0:20]                               <- message header
	// [20:20+m.Length]                     <- existing message attributes
	// [20+m.Length:20+m.Length+len(v) + 4] <- allocated buffer for new TLV
	// [first:last]                         <- same as previous
	// [0 1|2 3|4    4 + len(v)]            <- mapping for allocated buffer
	//   T   L        V
	allocSize := attributeHeaderSize + len(v)  // len(TLV) = len(TL) + len(V)
	first := messageHeaderSize + int(m.Length) // first byte number
	last := first + allocSize                  // last byte number
	m.grow(last)                               // growing cap(Raw) to fit TLV
	m.Raw = m.Raw[:last]                       // now len(Raw) = last
	m.Length += uint32(allocSize)              // rendering length change

	// Sub-slicing internal buffer to simplify encoding.
	buf := m.Raw[first:last]           // slice for TLV
	value := buf[attributeHeaderSize:] // slice for V
	attr := RawAttribute{
		Type:   t,              // T
		Length: uint16(len(v)), // L
		Value:  value,          // V
	}

	// Encoding attribute TLV to allocated buffer.
	bin.PutUint16(buf[0:2], attr.Type.Value()) // T
	bin.PutUint16(buf[2:4], attr.Length)       // L
	copy(value, v)                             // V

	// Checking that attribute value needs padding.
	if attr.Length%padding != 0 {
		// Performing padding.
		bytesToAdd := nearestPaddedValueLength(len(v)) - len(v)
		last += bytesToAdd
		m.grow(last)
		// setting all padding bytes to zero
		// to prevent data leak from previous
		// data in next bytesToAdd bytes
		buf = m.Raw[last-bytesToAdd : last]
		for i := range buf {
			buf[i] = 0
		}
		m.Raw = m.Raw[:last]           // increasing buffer length
		m.Length += uint32(bytesToAdd) // rendering length change
	}
	m.Attributes = append(m.Attributes, attr)
	m.WriteLength()
}

//初始化消息结构
func (m *Message) Reset() {
	m.Raw = m.Raw[:0]
	m.Length = 0
	m.Attributes = m.Attributes[:0]
}

func (m *Message) grow(v int) {
	n := len(m.Raw) + v
	for cap(m.Raw) < n {
		m.Raw = append(m.Raw, 0)
	}
	m.Raw = m.Raw[:n]
}

//写入消息长度到buff
func (m *Message) WriteLength() {
	_ = m.Raw[4]
	bin.PutUint16(m.Raw[2:4], uint16(m.Length))
}

//构建消息头部信息
func (m *Message) WriteHeader() {
	if len(m.Raw) < messageHeaderSize {
		//分配20字节头部空间
		m.grow(messageHeaderSize)
	}
	_ = m.Raw[:messageHeaderSize]

	//写入消息类型
	m.WriteType()
	//写入消息长度
	m.WriteLength()
	//写入magicCookie
	bin.PutUint32(m.Raw[4:8], magicCookie)
	//写入TransactionID
	copy(m.Raw[8:messageHeaderSize], m.TransactionID[:])
}

//构建消息结构
func (m *Message) Build(setters ...Setter) error {
	m.Reset()
	m.WriteHeader()
	for _, s := range setters {
		if err := s.AddTo(m); err != nil {
			return err
		}
	}
	return nil
}

//方法定义
type Method uint16

const (
	MethodBinding          Method = 0x001
	MethodAllocate         Method = 0x003
	MethodRefresh          Method = 0x004
	MethodSend             Method = 0x006
	MethodData             Method = 0x007
	MethodCreatePermission Method = 0x008
	MethodChannelBind      Method = 0x009
)

var methodName = map[Method]string{
	MethodBinding:          "binding",
	MethodAllocate:         "allocate",
	MethodRefresh:          "refresh",
	MethodSend:             "send",
	MethodData:             "data",
	MethodCreatePermission: "create permission",
	MethodChannelBind:      "channel bind",
}

func (m Method) String() string {
	s, ok := methodName[m]
	if !ok {
		// Falling back to hex representation.
		s = fmt.Sprintf("0x%x", uint16(m))
	}
	return s
}

const (
	methodABits  = 0xf   // 0b0000000000001111
	methodBBits  = 0x70  // 0b0000000001110000
	methodDBits  = 0xf80 // 0b0000111110000000
	methodBShift = 1
	methodDShift = 2

	firstBit  = 0x1
	secondBit = 0x2

	c0Bit = firstBit
	c1Bit = secondBit

	classC0Shift = 4
	classC1Shift = 7
)

// 0                 1
// 2  3  4 5 6 7 8 9 0 1 2 3 4 5
// +--+--+-+-+-+-+-+-+-+-+-+-+-+-+
// |M |M |M|M|M|C|M|M|M|C|M|M|M|M|
// |11|10|9|8|7|1|6|5|4|0|3|2|1|0|
// +--+--+-+-+-+-+-+-+-+-+-+-+-+-+
type MessageType struct {
	//Message的M0 - M11代表Method
	Method Method
	//Message的C0 - C1代表Class
	Class MessageClass
}

//v转换成MessageType类型
func (t *MessageType) ReadValue(v uint16) {
	//转换Class
	//C0位
	c0 := (v >> classC0Shift) & c0Bit
	//C1位
	c1 := (v >> classC1Shift) & c1Bit
	//0b00 = request
	//0b01 = indication
	//0b10 = response
	//0b11 = error response
	class := c0 + c1
	t.Class = MessageClass(class)

	//转换Method
	a := v & methodABits                   // A(M0-M3)
	b := (v >> methodBShift) & methodBBits // B(M4-M6)
	d := (v >> methodDShift) & methodDBits // D(M7-M11)
	m := a + b + d
	t.Method = Method(m)
}

func (t MessageType) AddTo(m *Message) error {
	m.SetType(t)
	return nil
}

//返回MessageType的字节表达形式
func (t MessageType) Value() uint16 {
	m := uint16(t.Method)
	a := m & methodABits // A = M * 0b0000000000001111 (right 4 bits)
	b := m & methodBBits // B = M * 0b0000000001110000 (3 bits after A)
	d := m & methodDBits // D = M * 0b0000111110000000 (5 bits after B)

	m = a + (b << methodBShift) + (d << methodDShift)

	c := uint16(t.Class)
	c0 := (c & c0Bit) << classC0Shift
	c1 := (c & c1Bit) << classC1Shift
	class := c0 + c1

	return m + class
}

type MessageClass byte

const (
	ClassRequest         MessageClass = 0x00 // 0b00
	ClassIndication      MessageClass = 0x01 // 0b01
	ClassSuccessResponse MessageClass = 0x02 // 0b10
	ClassErrorResponse   MessageClass = 0x03 // 0b11
)

//新建消息类型
func NewType(method Method, class MessageClass) MessageType {
	return MessageType{
		Method: method,
		Class:  class,
	}
}

// Common STUN message types.
var (
	// Bind请求消息类型
	BindingRequest = NewType(MethodBinding, ClassRequest)
	// Binding成功消息类型
	BindingSuccess = NewType(MethodBinding, ClassSuccessResponse)
	// Binding错误消息类型
	BindingError = NewType(MethodBinding, ClassErrorResponse)
)

func (c MessageClass) String() string {
	switch c {
	case ClassRequest:
		return "request"
	case ClassIndication:
		return "indication"
	case ClassSuccessResponse:
		return "success response"
	case ClassErrorResponse:
		return "error response"
	default:
		panic("unknown message class")
	}
}

const padding = 4

func nearestPaddedValueLength(l int) int {
	n := padding * (l / padding)
	if n < l {
		n += padding
	}
	return n
}
