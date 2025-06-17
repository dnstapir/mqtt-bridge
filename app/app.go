package app

import (
	"errors"
	"sync"

	"github.com/dnstapir/mqtt-bridge/app/downbridge"
	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/app/upbridge"
	"github.com/dnstapir/mqtt-bridge/shared"
)

type App struct {
	Log     shared.LoggerIF
	Mqtt    shared.MqttIF
	Nats    shared.NatsIF
	Nodeman shared.NodemanIF
	Bridges []Bridge

	isInitialized bool
	doneChan      chan error
	stopChan      chan bool
	wg            *sync.WaitGroup
}

type Bridge struct {
	Direction   string `toml:"Direction"`
	MqttTopic   string `toml:"MqttTopic"`
	NatsSubject string `toml:"NatsSubject"`
	NatsQueue   string `toml:"NatsQueue"`
	Key         string `toml:"Key"`
	Schema      string `toml:"Schema"`
}

func (a *App) Initialize() error {
	var wg sync.WaitGroup
	a.wg = &wg

	a.doneChan = make(chan error, 10)
	a.stopChan = make(chan bool, 1)

	if a.Log == nil {
		return errors.New("no logger object")
	}

	if a.Mqtt == nil {
		return errors.New("no mqtt object")
	}

	if a.Nats == nil {
		return errors.New("no nats object")
	}

	if a.Nodeman == nil {
		return errors.New("no nodeman object")
	}

	if len(a.Bridges) == 0 {
		return errors.New("no bridge configuration")
	}

	err := keys.SetLogger(a.Log)
	if err != nil {
		return err
	}

	a.isInitialized = true
	return nil
}

func (a *App) Run() <-chan error {
	if !a.isInitialized {
		panic("app not initialized")
	}

	a.Log.Info("Starting main loop")
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		err := a.Nats.Connect()
		if err != nil {
			a.doneChan <- err
			return
		}

		a.startBridges()
		a.Log.Info("Entering main loop")
		for {
			select {
			case <-a.stopChan:
				a.Log.Info("Stopping main worker thread")
				return
			}
		}
	}()

	a.Log.Info("Application is now up and running")
	return a.doneChan
}

func (a *App) Stop() error {
	if a.isInitialized {
		a.Log.Info("Stopping application")
	} else {
		a.Log.Info("Stop() called but application was not initialized")
	}

	a.stopChan <- true
	a.wg.Wait()

	close(a.doneChan)
	close(a.stopChan)

	a.Log.Info("Application stopped")

	return nil
}

func (a *App) startBridges() {
	for _, bridge := range a.Bridges {
		if bridge.Direction == "up" {
			conf := upbridge.Conf{
				Log:     a.Log,
				Nodeman: a.Nodeman,
				Key:     bridge.Key,
				Schema:  bridge.Schema,
			}
			ub, err := upbridge.Create(conf)
			if err != nil {
				panic(err)
			}

			inCh, err := a.Mqtt.Subscribe(bridge.MqttTopic)
			if err != nil {
				panic(err)
			}

			outCh, err := a.Nats.StartPublishing(bridge.NatsSubject, bridge.NatsQueue)
			if err != nil {
				panic(err)
			}

			go ub.Start(inCh, outCh)
		} else if bridge.Direction == "down" {
			conf := downbridge.Conf{
				Log:    a.Log,
				Key:    bridge.Key,
				Schema: bridge.Schema,
			}
			db, err := downbridge.Create(conf)
			if err != nil {
				panic(err)
			}

			inCh, err := a.Nats.Subscribe(bridge.NatsSubject, bridge.NatsQueue)
			if err != nil {
				panic(err)
			}

			outCh, err := a.Mqtt.StartPublishing(bridge.MqttTopic)
			if err != nil {
				panic(err)
			}

			go db.Start(inCh, outCh)
		} else {
			panic("unsupported bridge direction")
		}
	}
}
