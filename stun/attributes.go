package stun

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

type AttrType uint16

// Attributes from comprehension-required range (0x0000-0x7FFF).
const (
	AttrMappedAddress     AttrType = 0x0001 // MAPPED-ADDRESS
	AttrUsername          AttrType = 0x0006 // USERNAME
	AttrMessageIntegrity  AttrType = 0x0008 // MESSAGE-INTEGRITY
	AttrErrorCode         AttrType = 0x0009 // ERROR-CODE
	AttrUnknownAttributes AttrType = 0x000A // UNKNOWN-ATTRIBUTES
	AttrRealm             AttrType = 0x0014 // REALM
	AttrNonce             AttrType = 0x0015 // NONCE
	AttrXORMappedAddress  AttrType = 0x0020 // XOR-MAPPED-ADDRESS
)

// Attributes from comprehension-optional range (0x8000-0xFFFF).
const (
	AttrSoftware        AttrType = 0x8022 // SOFTWARE
	AttrAlternateServer AttrType = 0x8023 // ALTERNATE-SERVER
	AttrFingerprint     AttrType = 0x8028 // FINGERPRINT
)

// Attributes from RFC 5245 ICE.
const (
	AttrPriority       AttrType = 0x0024 // PRIORITY
	AttrUseCandidate   AttrType = 0x0025 // USE-CANDIDATE
	AttrICEControlled  AttrType = 0x8029 // ICE-CONTROLLED
	AttrICEControlling AttrType = 0x802A // ICE-CONTROLLING
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
