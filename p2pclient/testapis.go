package p2pclient

import (
	"github.com/cocobao/cocostun/stun"
	"github.com/cocobao/log"
)

func (c *P2PClient) TestI(callback func(res stun.AgentEvent)) {
	c.sendBindRequest(false, false, callback)
}

func (c *P2PClient) TestII(callback func(res stun.AgentEvent)) {
	c.sendBindRequest(true, true, callback)
}

func (c *P2PClient) TestIII(callback func(res stun.AgentEvent)) {
	c.sendBindRequest(false, true, callback)
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
func (c *P2PClient) Discover(f func()) {
	c.natType = NATError
	c.ChangeServerAddr(c.serverAddr)
	log.Debugf("----++++send testI %s ----++++", c.serverAddr)
	c.TestI(func(res stun.AgentEvent) {
		if res.Error != nil {
			log.Warn(res.Error)
			f()
			return
		}
		log.Debug("local:", c.localAddrStr)
		attInfos1 := res.Message.AsyncAttrbutes(c.localAddrStr)
		if attInfos1.MappedAddr != nil {
			c.mapAddrStr = attInfos1.MappedAddr
			log.Debug("map addr:", attInfos1.MappedAddr.String())
		}

		if attInfos1.OtherAddr != nil {
			log.Debug("other addr:", attInfos1.OtherAddr.String())
		}

		changedAddr := attInfos1.ChangedAddr
		if changedAddr == nil {
			changedAddr = attInfos1.OtherAddr
		}

		if changedAddr != nil {
			log.Debug("change addr:", attInfos1.ChangedAddr.String())
		} else {
			log.Warn("no change addr")
			f()
			return
		}

		log.Debugf("----++++send testII %s ----++++", c.serverAddr)
		c.TestII(func(res stun.AgentEvent) {
			if res.Error != nil {
				log.Warn(res.Error)
				if res.Error != stun.ErrTransactionTimeOut {
					f()
					return
				}
			}

			//本地ip跟nat映射ip一致情况
			if attInfos1.Identical {
				if res.Message == nil {
					c.natType = NATSymmetricUDPFirewall
					f()
					return
				}
				c.natType = NATNone
				return
			}
			if res.Message != nil {
				c.natType = NATFull
				f()
				return
			}

			//切换服务器ip
			c.ChangeServerAddr(changedAddr.String())
			log.Debugf("----++++send testI %s ----++++", changedAddr.String())
			c.TestI(func(res stun.AgentEvent) {
				if res.Error != nil {
					log.Warn(res.Error)
					if res.Error != stun.ErrTransactionTimeOut {
						f()
						return
					}
				}

				if res.Message == nil {
					c.natType = NATUnknown
					f()
					return
				}
				attInfos2 := res.Message.AsyncAttrbutes(c.localAddrStr)
				log.Debugf("recv:%+v", attInfos2)

				if c.mapAddrStr.IP() == attInfos1.MappedAddr.IP() && c.mapAddrStr.Port() == attInfos2.MappedAddr.Port() {
					log.Debugf("----++++send testIII %s----++++", changedAddr.String())
					c.TestIII(func(res stun.AgentEvent) {
						defer f()
						if res.Error != nil {
							log.Warn(res.Error)
							if res.Error != stun.ErrTransactionTimeOut {
								f()
								return
							}
						}

						if res.Message == nil {
							c.natType = NATPortRestricted
							return
						}
						log.Debug("recv")
						c.natType = NATRestricted
					})
				} else {
					c.natType = NATSymmetric
					f()
				}
			})
		})
	})
}
