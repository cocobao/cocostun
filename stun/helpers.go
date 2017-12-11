package stun

type (
	Setter interface {
		AddTo(m *Message) error
	}

	Getter interface {
		GetFrom(m *Message) error
	}

	Checker interface {
		Check(m *Message) error
	}
)

type transactionIDSetter struct{}

func (transactionIDSetter) AddTo(m *Message) error {
	return m.NewTransactionID()
}

var TransactionID Setter = transactionIDSetter{}

func MustBuild(setters ...Setter) *Message {
	m, err := Build(setters...)
	if err != nil {
		panic(err)
	}
	return m
}

func Build(setters ...Setter) (*Message, error) {
	m := new(Message)
	return m, m.Build(setters...)
}
