package app

import (
	"github.com/dnstapir/mqtt-sender/app/log"
	"github.com/dnstapir/mqtt-sender/app/mqtt"
	"github.com/dnstapir/mqtt-sender/app/nats"
)

type App struct {
	Debug            bool
	MqttUrl          string
	MqttTopic        string
	MqttCaCert       string
	MqttClientCert   string
	MqttClientKey    string
	MqttEnableTlsKlf bool
	MqttTlsKlfPath   string
	MqttSigningKey   string
	NatsUrl          string
	NatsStream       string
	NatsConsumer     string
	NatsSubject      string
}

func (a App) Run() {
	var err error

	log.Initialize(a.Debug)
	log.Info("Logging initialized")
	log.Debug("Debug enabled")

	/* Set up MQTT */
	mqttClient, err := mqtt.Create(
		mqtt.Url(a.MqttUrl),
		mqtt.Topic(a.MqttTopic),
		mqtt.CaCert(a.MqttCaCert),
		mqtt.ClientCert(a.MqttClientCert, a.MqttClientKey),
		mqtt.TlsKeylogfile(a.MqttEnableTlsKlf, a.MqttTlsKlfPath),
		mqtt.SigningKey(a.MqttSigningKey),
	)
	if err != nil {
		panic(err)
	}
	log.Info("MQTT client created")

	/* Set up NATS */
	natsListener, err := nats.Create(
		nats.Url(a.NatsUrl),
		nats.Stream(a.NatsStream),
		nats.Consumer(a.NatsConsumer),
		nats.Subject(a.NatsSubject),
		nats.Handler(mqttClient.Publish),
	)
	if err != nil {
		panic(err)
	}
	log.Info("NATS listener created")

	natsListener.Start()
	log.Info("NATS listener started")
}
