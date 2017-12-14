package stun

import "time"

const (
	fingerprint        = 0x5354554e
	magicCookie        = 0x2112A442 // magicCookie 固定值为0x2112A442
	defaultTimeoutRate = time.Millisecond * 100

	familyIPv4 uint16 = 0x01
	familyIPv6 uint16 = 0x02
)
