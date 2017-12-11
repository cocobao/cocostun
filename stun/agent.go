package stun

import (
	"errors"
	"sync"
	"time"
)

const (
	agentCollectCap = 100
)

var (
	ErrAgentClosed          = errors.New("agent is closed")
	ErrTransactionTimeOut   = errors.New("transaction is timed out")
	ErrTransactionExists    = errors.New("transaction exists with same id")
	ErrTransactionStopped   = errors.New("transaction is stopped")
	ErrTransactionNotExists = errors.New("transaction not exists")
)

type transactionID [TransactionIDSize]byte

type AgentOptions struct {
	Handler AgentFn // Default handler, can be nil.
}

func NewAgent(o AgentOptions) *Agent {
	a := &Agent{
		transactions: make(map[transactionID]agentTransaction),
		zeroHandler:  o.Handler,
	}
	return a
}

type AgentFn func(e AgentEvent)

type AgentEvent struct {
	Message *Message
	Error   error
}

type Agent struct {
	transactions map[transactionID]agentTransaction
	closed       bool       // all calls are invalid if true
	mux          sync.Mutex // protects transactions and closed
	zeroHandler  AgentFn    // handles non-registered transactions if set
}

func (a *Agent) StopWithError(id [TransactionIDSize]byte, err error) error {
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	t, exists := a.transactions[id]
	delete(a.transactions, id)
	a.mux.Unlock()
	if !exists {
		return ErrTransactionNotExists
	}
	t.f(AgentEvent{
		Error: err,
	})
	return nil
}

//停止事务
func (a *Agent) Stop(id [TransactionIDSize]byte) error {
	return a.StopWithError(id, ErrTransactionStopped)
}

//启动代理
func (a *Agent) Start(id [TransactionIDSize]byte, deadline time.Time, f AgentFn) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.closed {
		return ErrAgentClosed
	}
	_, exists := a.transactions[id]
	if exists {
		return ErrTransactionExists
	}
	a.transactions[id] = agentTransaction{
		id:       id,
		f:        f,
		deadline: deadline,
	}
	return nil
}

//处理超时的事务数据
func (a *Agent) Collect(gcTime time.Time) error {
	toCall := make([]AgentFn, 0, agentCollectCap)
	toRemove := make([]transactionID, 0, agentCollectCap)
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}

	for id, t := range a.transactions {
		//取出已经超时的事务
		if t.deadline.Before(gcTime) {
			toRemove = append(toRemove, id)
			toCall = append(toCall, t.f)
		}
	}

	//从缓存事务组里删除已经超时的事务
	for _, id := range toRemove {
		delete(a.transactions, id)
	}

	a.mux.Unlock()
	event := AgentEvent{
		Error: ErrTransactionTimeOut,
	}
	//对超时的事务进行超时回调处理
	for _, f := range toCall {
		f(event)
	}
	return nil
}

func (a *Agent) Process(m *Message) error {
	e := AgentEvent{
		Message: m,
	}
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	//根据TransactionID取出之前本地缓存事务，相当于会话缓存，并删除缓存
	t, ok := a.transactions[m.TransactionID]
	delete(a.transactions, m.TransactionID)
	a.mux.Unlock()
	if ok {
		//消息事务回调
		t.f(e)
	} else if a.zeroHandler != nil {
		a.zeroHandler(e)
	}
	return nil
}

//关闭处理
func (a *Agent) Close() error {
	e := AgentEvent{
		Error: ErrAgentClosed,
	}
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	for _, t := range a.transactions {
		//所有数据回调连接错误
		t.f(e)
	}
	//清除事务缓存
	a.transactions = nil
	a.closed = true
	a.zeroHandler = nil
	a.mux.Unlock()
	return nil
}

type agentTransaction struct {
	id       transactionID
	deadline time.Time
	f        AgentFn
}
