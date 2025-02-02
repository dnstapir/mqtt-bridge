package app

import (
    "os"
    "os/signal"
    "syscall"

    "tapir-core-mqtt-sender/app/log"
    "tapir-core-mqtt-sender/app/mqtt"
    "tapir-core-mqtt-sender/app/nats"
)

type App struct {
    debug bool
}

func Create(options ...func(*App) error) (*App, error) {
    app := new(App)

    for _, opt := range options {
        err := opt(app)
        if err != nil {
            panic(err)
        }
    }

    return app, nil
}

func (a App) Run() {
    var err error
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

    log.Initialize(a.debug)
    log.Info("Logging initialized")
    log.Debug("Debug enabled")

    /* Set up MQTT */
    err = mqtt.Initialize()
    if err != nil {
        panic(err)
    }
    log.Info("MQTT initialized")

    /* Set up NATS */
    natsConf := nats.Config{
        Url:     "localhost:4222",
        Bucket:  "observations",
        Subject: "tapircore.events.observations.>",
    }
    err = nats.Initialize(natsConf)
    if err != nil {
        panic(err)
    }
    log.Info("NATS listener initialized")

    nats.Start(mqtt.Publish)
    log.Info("NATS listener started")

    s := <-c
    log.Info("Got signal %s", s)
}

/*
 * HERE BE OPTIONS
 */

func Debug(debug bool) func(*App) error {
    fPtr := func(app *App) error {
        app.debug = debug
        return nil
    }

    return fPtr
}
