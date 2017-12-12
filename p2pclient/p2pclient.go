package p2pclient

import (
	"fmt"
	"time"

	"github.com/cocobao/cocostun/stun"
)

func NewP2PClient(serverAddr string, softwareName string) (*P2PClient, error) {
	sc, err := stun.Dial("udp", serverAddr)
	if err != nil {
		return nil, err
	}

	cli := &P2PClient{
		serverAddr:   serverAddr,
		softwareName: softwareName,
		sc:           sc,
	}

	return cli, nil
}

type P2PClient struct {
	serverAddr   string
	softwareName string
	sc           *stun.Client
}

func (c *P2PClient) SetSoftwareName(name string) {
	c.softwareName = name
}

//发送绑定请求
func (c *P2PClient) sendBindRequest(changeIP bool, changePort bool) {
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	message.AddSoftwareAttribute(c.softwareName)
	if changeIP || changePort {
		message.AddChangeReqAttribute(changeIP, changePort)
	}
	message.AddFingerprintAttribute()
	c.sc.Do(message, time.Now().Add(time.Second*5), func(res stun.AgentEvent) {
		fmt.Println(res.Message.Attributes)
	})
}

// Figure 2: Flow for type discovery process (from RFC 3489).
//                        +--------+
//                        |  Test  |
//                        |   I    |
//                        +--------+
//                             |
//                             |
//                             V
//                            /\              /\
//                         N /  \ Y          /  \ Y             +--------+
//          UDP     <-------/Resp\--------->/ IP \------------->|  Test  |
//          Blocked         \ ?  /          \Same/              |   II   |
//                           \  /            \? /               +--------+
//                            \/              \/                    |
//                                             | N                  |
//                                             |                    V
//                                             V                    /\
//                                         +--------+  Sym.      N /  \
//                                         |  Test  |  UDP    <---/Resp\
//                                         |   II   |  Firewall   \ ?  /
//                                         +--------+              \  /
//                                             |                    \/
//                                             V                     |Y
//                  /\                         /\                    |
//   Symmetric  N  /  \       +--------+   N  /  \                   V
//      NAT  <--- / IP \<-----|  Test  |<--- /Resp\               Open
//                \Same/      |   I    |     \ ?  /               Internet
//                 \? /       +--------+      \  /
//                  \/                         \/
//                  |Y                          |Y
//                  |                           |
//                  |                           V
//                  |                           Full
//                  |                           Cone
//                  V              /\
//              +--------+        /  \ Y
//              |  Test  |------>/Resp\---->Restricted
//              |   III  |       \ ?  /
//              +--------+        \  /
//                                 \/
//                                  |N
//                                  |       Port
//                                  +------>Restricted
