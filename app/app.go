package app

import (
    "tapir-core-mqtt-sender/app/log"
    "tapir-core-mqtt-sender/app/mqtt"
    "tapir-core-mqtt-sender/app/nats"
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
    NatsBucket       string
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
        nats.Bucket(a.NatsBucket),
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
