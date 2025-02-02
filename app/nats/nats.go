package nats

import (
    "context"
    "time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

    "tapir-core-mqtt-sender/app/log"
)

type Config struct {
    Url string
    Bucket string
    Subject string
}

type HandlerFunc func([]byte) error

var bucketCh <-chan jetstream.KeyValueEntry

func Initialize(conf Config) error {
    nc, err := nats.Connect(nats.DefaultURL)

    if err != nil {
        panic(err)
    }

    js, err := jetstream.New(nc)
    if err != nil {
        panic(err)
    }

    ctx, _ := context.WithTimeout(context.Background(), 100*time.Second)

    kv, err := js.KeyValue(ctx, conf.Bucket)
    if err != nil {
        panic(err)
    }

    w, err := kv.Watch(ctx, conf.Subject)
    if err != nil {
        panic(err)
    }

    bucketCh = w.Updates()
    return nil
}

func Start(handler HandlerFunc) error {
    if bucketCh == nil {
        panic("NATS listener not initialized")
    }

    go func() {
        for u := range bucketCh {
            if u != nil {
                log.Debug("Data received on subject '%s'", u.Key())
                err := handler(u.Value())
                if err != nil {
                    panic(err)
                }
            }
        }
    }()

    return nil
}
