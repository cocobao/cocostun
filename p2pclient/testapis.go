package p2pclient

func (c *P2PClient) TestI() {
	c.sendBindRequest(false, false)
}

func (c *P2PClient) TestII() {
	c.sendBindRequest(true, true)
}

func (c *P2PClient) TestIII() {
	c.sendBindRequest(false, true)
}
