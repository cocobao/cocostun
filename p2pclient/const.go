package p2pclient

type NATType int

// NAT types.
const (
	NATError NATType = iota
	NATUnknown
	NATNone
	NATBlocked
	NATFull
	NATSymmetric
	NATRestricted
	NATPortRestricted
	NATSymmetricUDPFirewall

	NATSymetric            = NATSymmetric
	NATSymetricUDPFirewall = NATSymmetricUDPFirewall
)

var natStr map[NATType]string

func init() {
	natStr = map[NATType]string{
		NATError:                "Test failed",
		NATUnknown:              "Unexpected response from the STUN server",
		NATBlocked:              "UDP is blocked",
		NATFull:                 "Full cone NAT",
		NATSymmetric:            "Symmetric NAT",
		NATRestricted:           "Restricted NAT",
		NATPortRestricted:       "Port restricted NAT",
		NATNone:                 "Not behind a NAT",
		NATSymmetricUDPFirewall: "Symmetric UDP firewall",
	}
}

func (nat NATType) String() string {
	if s, ok := natStr[nat]; ok {
		return s
	}
	return "Unknown"
}