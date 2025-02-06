package nats

import (
	"context"
	"errors"
    "strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dnstapir/mqtt-sender/app/log"
)

type HandlerFunc func(string, []byte) error

type Nats struct {
	url           string
	subject       string
    stream        string
    consumer      string
	handler       HandlerFunc
    messagesCh    jetstream.MessagesContext
	started       bool
	ctx           context.Context
}

func Create(options ...func(*Nats) error) (*Nats, error) {
	n := new(Nats)

	for _, opt := range options {
		err := opt(n)
		if err != nil {
			panic(err)
		}
	}

	nc, err := nats.Connect(n.url)
	if err != nil {
		panic(err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	n.ctx, _ = context.WithTimeout(context.Background(), 100*time.Second)

    stream, err := js.CreateStream(n.ctx, jetstream.StreamConfig{
        Name: n.stream,
        Subjects: []string{n.subject},
    })
	if err != nil {
		panic(err)
	}

    consumer, err := stream.CreateConsumer(n.ctx, jetstream.ConsumerConfig{
        Durable: n.consumer,
        AckPolicy: jetstream.AckExplicitPolicy,
    })
	if err != nil {
		panic(err)
	}

    messagesCh, err := consumer.Messages()
	if err != nil {
		panic(err)
	}

    n.messagesCh = messagesCh

	return n, nil
}

func (n *Nats) Start() error {
	if n.messagesCh == nil {
		panic("NATS listener not initialized")
	}

	n.started = true

	go func() {
		log.Debug("NATS handler thread spawned")
        for {
            msg, err := n.messagesCh.Next()
            if err != nil {
                panic(err)
            }

            data := strings.Fields(string(msg.Data()))

            // TODO validate incoming data

			err = n.handler(data[0], []byte(data[1]))
            if err != nil {
                panic(err)
            }

            msg.Ack()
        }

		n.started = false
	}()

	return nil
}

func Url(url string) func(*Nats) error {
	fptr := func(n *Nats) error {
		if n.started {
			return errors.New("Error configuring url, client already started")
		}
		n.url = url

		log.Info("NATS URL configured: '%s'", n.url)
		return nil
	}

	return fptr
}

func Stream(stream string) func(*Nats) error {
	fptr := func(n *Nats) error {
		if n.started {
			return errors.New("Error configuring stream, client already started")
		}
		n.stream = stream

		log.Info("NATS stream configured: '%s'", n.stream)
		return nil
	}

	return fptr
}

func Consumer(consumer string) func(*Nats) error {
	fptr := func(n *Nats) error {
		if n.started {
			return errors.New("Error configuring consumer, client already started")
		}
		n.consumer = consumer

		log.Info("NATS consumer configured: '%s'", n.consumer)
		return nil
	}

	return fptr
}

func Subject(subject string) func(*Nats) error {
	fptr := func(n *Nats) error {
		if n.started {
			return errors.New("Error configuring subject, client already started")
		}
		n.subject = subject

		log.Info("NATS subject configured: '%s'", n.subject)
		return nil
	}

	return fptr
}

func Handler(handler HandlerFunc) func(*Nats) error {
	fptr := func(n *Nats) error {
		if n.started {
			return errors.New("Error configuring handler, client already started")
		}

		n.handler = handler
		log.Info("NATS handler configured")
		return nil
	}

	return fptr
}
