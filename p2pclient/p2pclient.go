package p2pclient

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cocobao/cocostun/stun"
)

func NewP2PClient(server string, softwareName string) (*P2PClient, error) {
	serverUDPAddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, fmt.Errorf("Resolve server addr fail")
	}
	sc, err := stun.Dial("udp", serverUDPAddr)
	if err != nil {
		return nil, err
	}

	var la string
	if c, err := net.Dial("udp", server); err == nil {
		la = c.LocalAddr().String()
		c.Close()

		scad := sc.LocalAddr()
		adss := strings.Split(scad, ":")
		if len(adss) > 0 {
			laadss := strings.Split(la, ":")
			la = fmt.Sprintf("%s:%s", laadss[0], adss[len(adss)-1])
		} else {
			la = scad
		}
	}

	cli := &P2PClient{
		serverHost:   server,
		serverAddr:   serverUDPAddr.String(),
		softwareName: softwareName,
		sc:           sc,
		localAddrStr: la,
	}

	return cli, nil
}

type P2PClient struct {
	serverHost   string
	serverAddr   string
	softwareName string
	sc           *stun.Client
	localAddrStr string
	mapAddrStr   *stun.Host
	natType      NATType
}

func (c *P2PClient) ChangeServerAddr(addr string) {
	c.sc.ChangeServerAddr(addr)
}

func (c *P2PClient) SetSoftwareName(name string) {
	c.softwareName = name
}

func (c *P2PClient) GetNatType() string {
	return c.natType.String()
}

//发送绑定请求
func (c *P2PClient) sendBindRequest(changeIP bool, changePort bool, callback func(res stun.AgentEvent)) {
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	message.AddSoftwareAttribute(c.softwareName)
	if changeIP || changePort {
		message.AddChangeReqAttribute(changeIP, changePort)
	}
	message.AddFingerprintAttribute()
	err := c.sc.SendMessage(message, time.Now().Add(time.Second*3), callback)
	if err != nil {
		callback(stun.AgentEvent{
			Error: err,
		})
	}
}
