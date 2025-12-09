package fake

import (
	"github.com/dnstapir/mqtt-bridge/shared"
)

type nats struct {
	subCh chan []byte
	pubCh chan shared.NatsData
}

func Nats() *nats {
	nats := new(nats)
	nats.subCh = make(chan []byte, 1)
	nats.pubCh = make(chan shared.NatsData)

	return nats
}
func (n *nats) Connect() error {
	return nil
}

func (n *nats) Subscribe(subject string, queue string) (<-chan []byte, error) {
	return n.subCh, nil
}

func (n *nats) Stop() {
}

func (n *nats) Inject(data []byte) {
	n.subCh <- data
}

func (n *nats) StartPublishing(subject string, queue string) (chan<- shared.NatsData, error) {
	return n.pubCh, nil
}

func (n *nats) Eavesdrop() shared.NatsData {
	data := <-n.pubCh
	return data
}
