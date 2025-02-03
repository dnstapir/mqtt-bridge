package nats

import (
    "context"
    "errors"
    "time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

    "tapir-core-mqtt-sender/app/log"
)

type HandlerFunc func([]byte) error

type Nats struct {
    url string
    bucket string
    subject string
    handler HandlerFunc
    bucketCh <-chan jetstream.KeyValueEntry
    started bool
    ctx context.Context
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

    kv, err := js.KeyValue(n.ctx, n.bucket)
    if err != nil {
        panic(err)
    }

    w, err := kv.Watch(n.ctx, n.subject)
    if err != nil {
        panic(err)
    }

    n.bucketCh = w.Updates()

    return n, nil
}

func (n *Nats) Start() error {
    if n.bucketCh == nil {
        panic("NATS listener not initialized")
    }

    n.started = true

    go func() {
        log.Debug("NATS handler thread spawned")
        for u := range n.bucketCh {
            if u != nil {
                log.Debug("Data received on subject '%s'", u.Key())
                err := n.handler(u.Value())
                if err != nil {
                    panic(err)
                }
            }
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

func Bucket(bucket string) func(*Nats) error {
    fptr := func(n *Nats) error {
        if n.started {
            return errors.New("Error configuring bucket, client already started")
        }
        n.bucket = bucket

        log.Info("NATS bucket configured: '%s'", n.bucket)
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
