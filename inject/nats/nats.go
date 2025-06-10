package nats

import (
    "errors"
	"github.com/dnstapir/mqtt-bridge/shared"
)

type Conf struct {
	Log      shared.ILogger
	NatsUrl  string
}

type natsclient struct {
}

func Create(conf Conf) (*natsclient, error) {
    return nil, errors.New("not implemented")
}

func (nc *natsclient) Subscribe(subject string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (nc *natsclient) StartPublishing(subject string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
