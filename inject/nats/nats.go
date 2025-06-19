package nats

import (
	"errors"
	"github.com/dnstapir/mqtt-bridge/shared"
	"github.com/nats-io/nats.go"
)

const cHEADER_DNSTAPIR_MESSAGE_SCHEMA = "DNSTAPIR-Message-Schema"
const cHEADER_DNSTAPIR_MQTT_TOPIC = "DNSTAPIR-Mqtt-Topic"
const cHEADER_DNSTAPIR_KEY_IDENTIFIER = "DNSTAPIR-Key-Identifier"
const cHEADER_DNSTAPIR_KEY_THUMBPRINT = "DNSTAPIR-Key-Thumbprint"

type Conf struct {
	Log     shared.LoggerIF
	NatsUrl string
}

type natsclient struct {
	url  string
	log  shared.LoggerIF
	conn *nats.Conn
}

func Create(conf Conf) (*natsclient, error) {
	newClient := new(natsclient)

	newClient.url = conf.NatsUrl
	newClient.log = conf.Log

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
	msgCh := make(chan *nats.Msg)
	byteCh := make(chan []byte)

    _, err := c.conn.ChanQueueSubscribe(subject, queue, msgCh)
	if err != nil {
        return nil, err
	}

    go func(){
	    for msg := range msgCh {
            c.log.Debug("Received message %s", string(msg.Data))
            byteCh <- msg.Data
		    msg.Ack()
        }
        c.log.Warning("Channel closed for subscription to '%s'", subject)
    }()

	return byteCh, nil
}

func (c *natsclient) StartPublishing(subject string, queue string) (chan<- []byte, error) {
	return nil, errors.New("nats.StartPublishing not implemented")
}
