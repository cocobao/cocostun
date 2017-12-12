package stun

type (
	Setter interface {
		AddTo(m *Message) error
	}

	Getter interface {
		GetFrom(m *Message) error
	}
)

func MustBuild(setters ...Setter) *Message {
	setters = append(setters, TransactionID)
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

var TransactionID Setter = transactionIDSetter{}

type transactionIDSetter struct{}

func (transactionIDSetter) AddTo(m *Message) error {
	return m.NewTransactionID()
}
