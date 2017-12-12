package p2pclient_test

import (
	"fmt"
	"testing"

	"github.com/cocobao/cocostun/p2pclient"
)

func TestClient(t *testing.T) {
	cli, err := p2pclient.NewP2PClient("stun.ekiga.net:3478", "cocosp2p")
	if err != nil {
		fmt.Println("new client fail, err:", err)
		return
	}
	cli.TestI()
}
