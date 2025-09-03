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
    subscriptionOutCh chan []byte
    sub *nats.Subscription
}

func Create(conf Conf) (*natsclient, error) {
	newClient := new(natsclient)

    newClient.subscriptionOutCh = make(chan []byte)

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
    sub, err := c.conn.QueueSubscribe(subject, queue, c.subscriptionCb)
	if err != nil {
        return nil, err
	}

    c.sub = sub
    c.log.Debug("Nats subscription done")

	return c.subscriptionOutCh, nil
}

func (c *natsclient) Stop() {
    c.sub.Unsubscribe()
    c.conn.Drain()
    close(c.subscriptionOutCh)
}

func (c *natsclient) StartPublishing(subject string, queue string) (chan<- []byte, error) {
    dataChan := make(chan []byte)

    go func(){
	    for data := range dataChan {
            msg := nats.NewMsg(subject)
            msg.Data = data
            // TODO NATS headers
            c.log.Debug("Attempting to publish NATS message %s", string(msg.Data))
            err := c.conn.PublishMsg(msg)
            if err != nil {
                c.log.Error("Failed to publish NATS message on subject '%s'", subject)
            }
            c.log.Debug("Published NATS message to subject %s!", subject)
        }
    }()

    return dataChan, nil
}

func (c *natsclient) subscriptionCb(msg *nats.Msg) {
    c.log.Debug("Received nats message %s", string(msg.Data))
    data := msg.Data
	msg.Ack()
    go func(){c.subscriptionOutCh <- data}()
    c.log.Debug("Done processing nats message")
}
