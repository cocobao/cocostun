package stun

import (
	"hash/crc32"
	"net"

	"github.com/cocobao/cocostun/utils"
)

type Attributes []RawAttribute

func (a Attributes) Get(t AttrType) (RawAttribute, bool) {
	for _, candidate := range a {
		if candidate.Type == t {
			return candidate, true
		}
	}
	return RawAttribute{}, false
}

//属性协议结构
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |         Type                  |            Length             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Value (variable)                ....
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type RawAttribute struct {
	Type   AttrType
	Length uint16 // ignored while encoding
	Value  []byte
}

//      0                   1                   2                   3
//      0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//     |x x x x x x x x|    Family     |         X-Port                |
//     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//     |                X-Address (Variable)
//     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
//             Figure 6: Format of XOR-MAPPED-ADDRESS Attribute
func (v *RawAttribute) xorAddr(transID []byte) *Host {
	xorIP := make([]byte, 16)
	for i := 0; i < len(v.Value)-4; i++ {
		xorIP[i] = v.Value[i+4] ^ transID[i]
	}
	family := uint16(v.Value[1])
	port := bin.Uint16(v.Value[2:4])
	// Truncate if IPv4, otherwise net.IP sometimes renders it as an IPv6 address.
	if family == familyIPv4 {
		xorIP = xorIP[:4]
	}
	x := bin.Uint16(transID[:2])
	return &Host{family, net.IP(xorIP).String(), port ^ x}
}

//       0                   1                   2                   3
//       0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//      |0 0 0 0 0 0 0 0|    Family     |           Port                |
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//      |                                                               |
//      |                 Address (32 bits or 128 bits)                 |
//      |                                                               |
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
//               Figure 5: Format of MAPPED-ADDRESS Attribute
func (v *RawAttribute) rawAddr() *Host {
	host := new(Host)
	host.family = uint16(v.Value[1])
	host.port = bin.Uint16(v.Value[2:4])
	// Truncate if IPv4, otherwise net.IP sometimes renders it as an IPv6 address.
	if host.family == familyIPv4 {
		v.Value = v.Value[:8]
	}
	host.ip = net.IP(v.Value[4:]).String()
	return host
}

type AttrType uint16

func (t AttrType) Value() uint16 {
	return uint16(t)
}

// Attributes from comprehension-required range (0x0000-0x7FFF).
const (
	AttrMappedAddress          AttrType = 0x0001 // MAPPED-ADDRESS
	AttrResponseAddress        AttrType = 0x0002
	AttrChangeRequest          AttrType = 0x0003
	AttrSourceAddress          AttrType = 0x0004
	AttrChangedAddress         AttrType = 0x0005
	AttrUsername               AttrType = 0x0006 // USERNAME
	AttrPassword               AttrType = 0x0007
	AttrMessageIntegrity       AttrType = 0x0008 // MESSAGE-INTEGRITY
	AttrErrorCode              AttrType = 0x0009 // ERROR-CODE
	AttrUnknownAttributes      AttrType = 0x000A // UNKNOWN-ATTRIBUTES
	AttrReflectedFrom          AttrType = 0x000b
	AttrBandwidth              AttrType = 0x0010
	AttrXorPeerAddress         AttrType = 0x0012
	AttrRealm                  AttrType = 0x0014 // REALM
	AttrNonce                  AttrType = 0x0015 // NONCE
	AttrXorRelayedAddress      AttrType = 0x0016
	AttrRequestedAddressFamily AttrType = 0x0017
	AttrXORMappedAddress       AttrType = 0x0020 // XOR-MAPPED-ADDRESS
	AttrTimerVal               AttrType = 0x0021
	AttrPadding                AttrType = 0x0026
	AttrResponsePort           AttrType = 0x0027
	AttrConnectionID           AttrType = 0x002a
)

// Attributes from comprehension-optional range (0x8000-0xFFFF).
const (
	AttrXorMappedAddressExp AttrType = 0x8020
	AttrSoftware            AttrType = 0x8022 // SOFTWARE
	AttrAlternateServer     AttrType = 0x8023 // ALTERNATE-SERVER
	AttrFingerprint         AttrType = 0x8028 // FINGERPRINT
)

// Attributes from RFC 5245 ICE.
const (
	AttrPriority       AttrType = 0x0024 // PRIORITY
	AttrUseCandidate   AttrType = 0x0025 // USE-CANDIDATE
	AttrICEControlled  AttrType = 0x8029 // ICE-CONTROLLED
	AttrICEControlling AttrType = 0x802A // ICE-CONTROLLING
)

const (
	AttrResponseOrigin = 0x802b
	AttrOtherAddress   = 0x802c
	AttrEcnCheckStun   = 0x802d
	AttrCiscoFlowdata  = 0xc000
)

// Attributes from RFC 5766 TURN.
const (
	AttrChannelNumber      AttrType = 0x000C // CHANNEL-NUMBER
	AttrLifetime           AttrType = 0x000D // LIFETIME
	AttrXORPeerAddress     AttrType = 0x0012 // XOR-PEER-ADDRESS
	AttrData               AttrType = 0x0013 // DATA
	AttrXORRelayedAddress  AttrType = 0x0016 // XOR-RELAYED-ADDRESS
	AttrEvenPort           AttrType = 0x0018 // EVEN-PORT
	AttrRequestedTransport AttrType = 0x0019 // REQUESTED-TRANSPORT
	AttrDontFragment       AttrType = 0x001A // DONT-FRAGMENT
	AttrReservationToken   AttrType = 0x0022 // RESERVATION-TOKEN
)

//添加软件名称属性
func (m *Message) AddSoftwareAttribute(name string) {
	m.Add(AttrSoftware, []byte(name))
}

//添加指纹属性
func (m *Message) AddFingerprintAttribute() {
	crc := crc32.ChecksumIEEE(m.Raw) ^ fingerprint
	buf := make([]byte, 4)
	bin.PutUint32(buf, crc)
	m.Add(AttrSoftware, []byte(buf))
}

//添加切换端口或ip请求
func (m *Message) AddChangeReqAttribute(changeIP bool, changePort bool) {
	value := make([]byte, 4)
	if changeIP {
		value[3] |= 0x04
	}
	if changePort {
		value[3] |= 0x02
	}
	m.Add(AttrChangeRequest, value)
}

type AttrInfos struct {
	// ServerAddr  *Host // 服务器端地址
	ChangedAddr *Host
	MappedAddr  *Host //  external addr of client NAT
	OtherAddr   *Host // to replace changedAddr in RFC 5780
	Identical   bool  // nat的映射地址是否跟本地地址一样
}

//分析属性
func (m *Message) AsyncAttrbutes(localAddr string) *AttrInfos {
	infos := &AttrInfos{}
	var (
		mappedAddr  *Host
		changedAddr *Host
		otherAddr   *Host
	)
	for _, attr := range m.Attributes {
		switch attr.Type {
		//经过异或处理的外部映射地址
		case AttrXORMappedAddress:
			mappedAddr = attr.xorAddr(m.Raw[4:20])
		case AttrXorMappedAddressExp:
			mappedAddr = attr.xorAddr(m.Raw[4:20])
		case AttrChangedAddress:
			ca := attr.rawAddr()
			if ca != nil {
				changedAddr = newHostFromStr(ca.String())
			}
		case AttrOtherAddress:
			ca := attr.rawAddr()
			if ca != nil {
				otherAddr = newHostFromStr(ca.String())
			}
		}
	}

	if mappedAddr != nil {
		infos.MappedAddr = mappedAddr
		infos.Identical = utils.IsLocalAddress(localAddr, mappedAddr.String())
	}

	if changedAddr != nil {
		infos.ChangedAddr = changedAddr
	}

	if otherAddr != nil {
		infos.OtherAddr = otherAddr
	}

	return infos
}
