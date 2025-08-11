package fake

type nats struct {
	subCh chan []byte
	pubCh chan []byte
}

func Nats() *nats {
	nats := new(nats)
	nats.subCh = make(chan []byte, 1)
	nats.pubCh = make(chan []byte)

	return nats
}
func (n *nats) Connect() error {
	return nil
}

func (n *nats) Subscribe(subject string, queue string) (<-chan []byte, error) {
	return n.subCh, nil
}

func (n *nats) Inject(data []byte) {
	n.subCh <- data
}

func (n *nats) StartPublishing(subject string, queue string) (chan<- []byte, error) {
	return n.pubCh, nil
}

func (n *nats) Eavesdrop() []byte {
	data := <-n.pubCh
	return data
}
