package app

import (
	"github.com/dnstapir/mqtt-bridge/app/log"
	"github.com/dnstapir/mqtt-bridge/app/bridge"
)

type App struct {
	Debug            bool
	MqttUrl          string
	MqttCaCert       string
	MqttClientCert   string
	MqttClientKey    string
	MqttEnableTlsKlf bool
	MqttTlsKlfPath   string
    NatsUrl          string
    Bridges          []Bridge
}

type Bridge struct {
    Direction     string
    MqttTopic     string
    NatsSubject   string
    NatsQueue     string
    Key           string
    Schema        string
}

func (a App) Run() {
	log.Initialize(a.Debug)
	log.Info("Logging initialized")
	log.Debug("Debug enabled")

    for i, b := range a.Bridges {
        newBridge, err := bridge.Create(b.Direction,
		    bridge.MqttUrl(a.MqttUrl),
		    bridge.CaCert(a.MqttCaCert),
		    bridge.ClientCert(a.MqttClientCert, a.MqttClientKey),
		    bridge.TlsKeylogfile(a.MqttEnableTlsKlf, a.MqttTlsKlfPath),
		    bridge.NatsUrl(a.NatsUrl),
		    bridge.Topic(b.MqttTopic),
		    bridge.DataKey(b.Key),
		    bridge.Subject(b.NatsSubject),
		    bridge.Queue(b.NatsQueue),
		    bridge.Schema(b.Schema),
        )
	    if err != nil {
	    	panic(err)
	    }
	    log.Info("Bridge %d created", i)

        err = newBridge.Start()
	    if err != nil {
	    	panic(err)
	    }
	    log.Info("Bridge %d started", i)
    }
}
