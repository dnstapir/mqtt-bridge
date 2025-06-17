package nats

import (
    "errors"
	"github.com/dnstapir/mqtt-bridge/shared"
	"github.com/nats-io/nats.go"
)

type Conf struct {
	Log      shared.LoggerIF
	NatsUrl  string
}

type natsclient struct {
    url    string
	conn   *nats.Conn
}

func Create(conf Conf) (*natsclient, error) {
    newClient := new(natsclient)

    newClient.url = conf.NatsUrl

    return newClient, nil
}

func (c *natsclient) Connect() error {
    if c.conn != nil {
        return errors.New("already has connection")
    }

	natsConn, err := nats.Connect(c.url)

	if err != nil {
		return err
	}
	c.conn = natsConn

    return nil
}

func (c *natsclient) Subscribe(subject string, queue string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (c *natsclient) StartPublishing(subject string, queue string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
