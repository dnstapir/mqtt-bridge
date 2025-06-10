package app

import (
	"errors"

	"github.com/dnstapir/mqtt-bridge/shared"
)

type App struct {
    Log           shared.ILogger
    Mqtt          iMqttClient
    Nats          iNatsClient
    Nodeman       iNodemanClient
	Bridges       []Bridge `toml:"Bridges"`

	isInitialized bool
	doneChan      chan error
}


type Bridge struct {
	Direction   string `toml:"Direction"`
	MqttTopic   string `toml:"MqttTopic"`
	NatsSubject string `toml:"NatsSubject"`
	NatsQueue   string `toml:"NatsQueue"`
	Key         string `toml:"Key"`
	Schema      string `toml:"Schema"`
}

type iMqttClient interface {
    Subscribe(string) (<-chan []byte, error)
    StartPublishing(string) (chan<- []byte, error)
}

type iNatsClient interface {
    Subscribe(string) (<-chan []byte, error)
    StartPublishing(string) (chan<- []byte, error)
}

type iNodemanClient interface {
}

func (a *App) Initialize() error {
	a.doneChan = make(chan error, 10)

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

	a.isInitialized = true
	return nil
}

func (a *App) Run() <-chan error {
	if !a.isInitialized {
		panic("app not initialized")
	}

	a.Log.Info("Starting main loop")
	go func() {
		a.Log.Info("did a work")
		a.doneChan <- nil
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

	close(a.doneChan)

	a.Log.Info("Application stopped")

	return nil
}

//func (a *App) Run() {
//	log.Initialize(a.Debug)
//	log.Info("Logging initialized")
//	log.Debug("Debug enabled")
//
//	mqttConf := mqtt.Conf{
//		Url:        a.MqttUrl,
//		CaCert:     a.MqttCaCert,
//		ClientCert: a.MqttClientCert,
//		ClientKey:  a.MqttClientKey,
//		Ctx:        a.Ctx,
//	}
//
//	if a.MqttEnableTlsKlf && a.MqttTlsKlfPath != "" {
//		mqttConf.Keylogfile = a.MqttTlsKlfPath + "_mqtt"
//	}
//
//	err := mqtt.Init(mqttConf)
//	if err != nil {
//		panic(err)
//	}
//
//	for i, b := range a.Bridges {
//		newBridge, err := bridge.Create(b.Direction, i+1000,
//			bridge.TlsKeylogfile(a.MqttEnableTlsKlf, a.MqttTlsKlfPath),
//			bridge.NatsUrl(a.NatsUrl),
//			bridge.Topic(b.MqttTopic),
//			bridge.DataKey(b.Key),
//			bridge.Subject(b.NatsSubject),
//			bridge.Queue(b.NatsQueue),
//			bridge.Schema(b.Schema),
//			bridge.NodemanApiUrl(a.NodemanApiUrl),
//		)
//		if err != nil {
//			panic(err)
//		}
//		log.Info("Bridge %d created", i)
//
//		if b.Direction == "up" {
//			err := mqtt.Subscribe(newBridge.Topic(), newBridge.IncomingPktHandler, newBridge.BridgeID())
//			if err != nil {
//				panic(err)
//			}
//		} else if b.Direction == "down" {
//			newBridge.SetPublishMethod(mqtt.Publish)
//		} else {
//			panic(errors.New("Invalid bridge mode configured"))
//		}
//
//		err = newBridge.Start()
//		if err != nil {
//			panic(err)
//		}
//		log.Info("Bridge %d started", i)
//	}
//}
