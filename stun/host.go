package stun

import (
	"net"
	"strconv"
)

type Host struct {
	family uint16
	ip     string
	port   uint16
}

func (h *Host) Family() uint16 {
	return h.family
}

func (h *Host) IP() string {
	return h.ip
}

func (h *Host) Port() uint16 {
	return h.port
}

func (h *Host) TransportAddr() string {
	return net.JoinHostPort(h.ip, strconv.Itoa(int(h.port)))
}

func (h *Host) String() string {
	return h.TransportAddr()
}

func newHostFromStr(s string) *Host {
	udpAddr, err := net.ResolveUDPAddr("udp", s)
	if err != nil {
		return nil
	}
	host := new(Host)
	if udpAddr.IP.To4() != nil {
		host.family = familyIPv4
	} else {
		host.family = familyIPv6
	}
	host.ip = udpAddr.IP.String()
	host.port = uint16(udpAddr.Port)
	return host
}
