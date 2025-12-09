package nats

import (
	"errors"
	"github.com/dnstapir/mqtt-bridge/shared"
	"github.com/nats-io/nats.go"
	"time"
)

type Conf struct {
	Log     shared.LoggerIF
	NatsUrl string
}

type natsclient struct {
	url               string
	log               shared.LoggerIF
	conn              *nats.Conn
	subscriptionOutCh chan []byte
	done              chan struct{}
	sub               *nats.Subscription
}

func Create(conf Conf) (*natsclient, error) {
	newClient := new(natsclient)

	newClient.subscriptionOutCh = make(chan []byte, 1024)
	newClient.done = make(chan struct{})

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
	if c.sub != nil {
		err := c.sub.Unsubscribe()
		if err != nil {
			c.log.Warning("Unsubscribe failed in nats: %s", err)
		}
		c.sub = nil
	}

	if c.conn != nil {
		err := c.conn.Drain()
		if err != nil {
			c.log.Warning("Drain failed in nats: %s", err)
		}
		c.conn = nil
	}

	close(c.done)
	time.Sleep(10 * time.Millisecond)
	close(c.subscriptionOutCh)
}

func (c *natsclient) StartPublishing(subject string, queue string) (chan<- shared.NatsData, error) {
	dataChan := make(chan shared.NatsData)

	go func() {
		for natsData := range dataChan {
			msg := nats.NewMsg(subject)
			msg.Data = natsData.Payload

			for _, h := range shared.NATSHEADERS_DNSTAPIR_ALL {
				val, ok := natsData.Headers[h]
				if ok {
					msg.Header.Add(h, val)
					c.log.Debug("Setting NATS header, '%s: %s'", h, val)
				}
			}

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

	go func() {
		select {
		case c.subscriptionOutCh <- msg.Data:
			c.log.Debug("Succesfully handled packet on subject '%s'", msg.Subject)
		case <-c.done:
			c.log.Warning("Shutdown signaled, aborting handling of incoming nats message")
			return
		}
	}()

	c.log.Debug("Done processing nats message")
}
