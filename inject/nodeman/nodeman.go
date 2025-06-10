package nodeman

import (
    "errors"
	"github.com/dnstapir/mqtt-bridge/shared"
)

type Conf struct {
	Log           shared.ILogger
	NodemanApiUrl string
}

type nodemanclient struct {
}

func Create(conf Conf) (*nodemanclient, error) {
    return nil, errors.New("not implemented")
}

func (nc *nodemanclient) Subscribe(subject string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (nc *nodemanclient) StartPublishing(subject string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
