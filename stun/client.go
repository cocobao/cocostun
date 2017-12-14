package stun

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	ErrClientClosed = errors.New("client is closed")
)

func Dial(network string, addr *net.UDPAddr) (*Client, error) {
	if addr == nil {
		return nil, fmt.Errorf("server address is nil")
	}

	// conn, err := net.Dial(network, address)
	conn, err := net.ListenUDP(network, nil)
	if err != nil {
		return nil, err
	}

	cli := NewClient(conn, addr)
	return cli, nil
}

// type ClientOptions struct {
// 	Agent      ClientAgent
// 	Connection Connection

// 	//默认100ms
// 	TimeoutRate time.Duration
// }

// type Connection interface {
// 	io.Reader
// 	io.Writer
// 	io.Closer
// }

//新建客户端
func NewClient(conn net.PacketConn, addr net.Addr) *Client {
	c := &Client{
		close: make(chan struct{}),
		// c:      options.Connection,
		// a:      options.Agent,
		// gcRate: options.TimeoutRate,

		serConn:      conn,
		serAddr:      addr,
		localAddrStr: conn.LocalAddr().String(),
	}
	if c.a == nil {
		c.a = NewAgent(AgentOptions{})
	}
	if c.gcRate == 0 {
		c.gcRate = defaultTimeoutRate
	}
	fmt.Println("local:", c.localAddrStr)
	c.wg.Add(2)
	go c.readUntilClosed()
	go c.collectUntilClosed()
	return c
}

type Client struct {
	a ClientAgent
	// c            Connection
	close        chan struct{}
	closed       bool
	closedMux    sync.RWMutex
	gcRate       time.Duration
	wg           sync.WaitGroup
	localAddrStr string

	serConn net.PacketConn
	serAddr net.Addr
}

func (c *Client) LocalAddr() string {
	return c.localAddrStr
}

func (c *Client) ChangeServerAddr(addr string) error {
	serverUDPAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	c.serAddr = serverUDPAddr
	return nil
}

//读数据协程
func (c *Client) readUntilClosed() {
	defer c.wg.Done()

	for {
		select {
		//关闭通知
		case <-c.close:
			return
		default:
		}

		var (
			n   int
			err error
		)
		//读数据
		m := new(Message)
		m.Raw = make([]byte, 1024)
		tBuf := m.Raw[:cap(m.Raw)]
		n, _, err = c.serConn.ReadFrom(tBuf)
		if err == nil {
			if err = m.Decode(); err != nil {
				fmt.Println("decode fail,err:", err)
			} else {
				//数据处理
				if pErr := c.a.Process(m); pErr == ErrAgentClosed {
					return
				}
			}
		} else {
			if n == 0 {
				fmt.Println("net close by peer")
			}
			fmt.Println("read invalid,", n)
		}
	}
}

//定时检测事务超时
func (c *Client) collectUntilClosed() {
	t := time.NewTicker(c.gcRate)
	defer t.Stop()
	defer c.wg.Done()

	for {
		select {
		case <-c.close:
			return
		case gcTime := <-t.C:
			err := c.a.Collect(gcTime)
			if err != nil && err != ErrAgentClosed {
				fmt.Println(err)
				return
			}
		}
	}
}

//关闭连接
func (c *Client) Close() error {
	c.closedMux.Lock()
	if c.closed {
		c.closedMux.Unlock()
		return ErrClientClosed
	}
	c.closed = true
	c.closedMux.Unlock()
	agentErr := c.a.Close()
	connErr := c.serConn.Close()
	close(c.close)
	c.wg.Wait()
	if agentErr == nil && connErr == nil {
		return nil
	}
	return fmt.Errorf("agenterr:%v, connerr:%v", agentErr, connErr)
}

//启动发送事务
func (c *Client) Start(m *Message, d time.Time, f func(AgentEvent)) error {
	c.closedMux.RLock()
	closed := c.closed
	c.closedMux.RUnlock()
	if closed {
		return ErrClientClosed
	}
	if f != nil {
		if err := c.a.Start(m.TransactionID, d, f); err != nil {
			return err
		}
	}
	_, err := c.serConn.WriteTo(m.Raw, c.serAddr)
	if err != nil && f != nil {
		//发送失败，停止代理
		if stopErr := c.a.Stop(m.TransactionID); stopErr != nil {
			return fmt.Errorf("stopErr:%v, Cause:%v", stopErr, err)
		}
	}
	return err
}

func (c *Client) Indicate(m *Message) error {
	return c.Start(m, time.Time{}, nil)
}

//处理事务消息
func (c *Client) SendMessage(m *Message, d time.Time, f func(AgentEvent)) error {
	if f == nil {
		return c.Indicate(m)
	}
	// cond := sync.NewCond(new(sync.Mutex))
	// processed := false
	// wrapper := func(e AgentEvent) {
	// f(e)
	// cond.L.Lock()
	// processed = true
	// cond.Broadcast()
	// cond.L.Unlock()
	// }
	if err := c.Start(m, d, f); err != nil {
		return err
	}
	// cond.L.Lock()
	// //强同步请求
	// for !processed {
	// 	cond.Wait()
	// }
	// cond.L.Unlock()
	return nil
}

type ClientAgent interface {
	Process(*Message) error
	Close() error
	Start(id [TransactionIDSize]byte, deadline time.Time, f AgentFn) error
	Stop(id [TransactionIDSize]byte) error
	Collect(time.Time) error
}
