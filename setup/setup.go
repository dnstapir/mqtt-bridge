package setup

import (
	"github.com/dnstapir/mqtt-bridge/app"
	"github.com/dnstapir/mqtt-bridge/inject/logging"
	"github.com/dnstapir/mqtt-bridge/inject/mqtt"
	"github.com/dnstapir/mqtt-bridge/inject/nats"
	"github.com/dnstapir/mqtt-bridge/inject/nodeman"
)

type AppConf struct {
	Debug          bool         `toml:"Debug"`
	Quiet          bool         `toml:"Quiet"`
	MqttUrl        string       `toml:"MqttUrl"`
	MqttCaCert     string       `toml:"MqttCaCert"`
	MqttClientCert string       `toml:"MqttClientCert"`
	MqttClientKey  string       `toml:"MqttClientKey"`
	NatsUrl        string       `toml:"NatsUrl"`
	NodemanApiUrl  string       `toml:"NodemanApiUrl"`
	Bridges        []app.Bridge `toml:"Bridges"`
}

func BuildApp(conf AppConf) (*app.App, error) {
	log := logging.Create(conf.Debug, conf.Quiet)

	mqttConf := mqtt.Conf{
		Log:            log,
		MqttUrl:        conf.MqttUrl,
		MqttCaCert:     conf.MqttCaCert,
		MqttClientCert: conf.MqttClientCert,
		MqttClientKey:  conf.MqttClientKey,
	}
	mqttClient, err := mqtt.Create(mqttConf)
	if err != nil {
		log.Error("Error creating mqtt client")
		return nil, err
	}

	natsConf := nats.Conf{
		Log:     log,
		NatsUrl: conf.NatsUrl,
	}
	natsClient, err := nats.Create(natsConf)
	if err != nil {
		log.Error("Error creating nats client")
		return nil, err
	}

	nodemanConf := nodeman.Conf{
		Log:           log,
		NodemanApiUrl: conf.NodemanApiUrl,
	}
	nodemanClient, err := nodeman.Create(nodemanConf)
	if err != nil {
		log.Error("Error creating nodeman client")
		return nil, err
	}

	a := new(app.App)
	a.Log = log
	a.Mqtt = mqttClient
	a.Nats = natsClient
	a.Nodeman = nodemanClient
	a.Bridges = conf.Bridges

	return a, nil
}
