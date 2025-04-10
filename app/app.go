package app

import (
	"context"
	"errors"

	"github.com/dnstapir/mqtt-bridge/app/bridge"
	"github.com/dnstapir/mqtt-bridge/app/log"
	"github.com/dnstapir/mqtt-bridge/app/mqtt"
)

type App struct {
	Debug            bool     `toml:"Debug"`
	MqttUrl          string   `toml:"MqttUrl"`
	MqttCaCert       string   `toml:"MqttCaCert"`
	MqttClientCert   string   `toml:"MqttClientCert"`
	MqttClientKey    string   `toml:"MqttClientKey"`
	MqttEnableTlsKlf bool     `toml:"MqttEnableTlsKlf"` // TODO rename to EnableTlsKlf
	MqttTlsKlfPath   string   `toml:"MqttTlsKlfPath"`   // TODO rename to TlsKlfPath
	NatsUrl          string   `toml:"NatsUrl"`
	NodemanApiUrl    string   `toml:"NodemanApiUrl"`
	Bridges          []Bridge `toml:"Bridges"`
	Ctx              context.Context
}

type Bridge struct {
	Direction   string `toml:"Direction"`
	MqttTopic   string `toml:"MqttTopic"`
	NatsSubject string `toml:"NatsSubject"`
	NatsQueue   string `toml:"NatsQueue"`
	Key         string `toml:"Key"`
	Schema      string `toml:"Schema"`
}

func (a App) Run() {
	log.Initialize(a.Debug)
	log.Info("Logging initialized")
	log.Debug("Debug enabled")

	mqttConf := mqtt.Conf{
		Url:        a.MqttUrl,
		CaCert:     a.MqttCaCert,
		ClientCert: a.MqttClientCert,
		ClientKey:  a.MqttClientKey,
		Ctx:        a.Ctx,
	}

	if a.MqttEnableTlsKlf && a.MqttTlsKlfPath != "" {
		mqttConf.Keylogfile = a.MqttTlsKlfPath + "_mqtt"
	}

	err := mqtt.Init(mqttConf)
	if err != nil {
		panic(err)
	}

	for i, b := range a.Bridges {
		newBridge, err := bridge.Create(b.Direction, i+1000,
			bridge.TlsKeylogfile(a.MqttEnableTlsKlf, a.MqttTlsKlfPath),
			bridge.NatsUrl(a.NatsUrl),
			bridge.Topic(b.MqttTopic),
			bridge.DataKey(b.Key),
			bridge.Subject(b.NatsSubject),
			bridge.Queue(b.NatsQueue),
			bridge.Schema(b.Schema),
			bridge.NodemanApiUrl(a.NodemanApiUrl),
		)
		if err != nil {
			panic(err)
		}
		log.Info("Bridge %d created", i)

		if b.Direction == "up" {
			err := mqtt.Subscribe(newBridge.Topic(), newBridge.IncomingPktHandler, newBridge.BridgeID())
			if err != nil {
				panic(err)
			}
		} else if b.Direction == "down" {
			newBridge.SetPublishMethod(mqtt.Publish)
		} else {
			panic(errors.New("Invalid bridge mode configured"))
		}

		err = newBridge.Start()
		if err != nil {
			panic(err)
		}
		log.Info("Bridge %d started", i)
	}
}
