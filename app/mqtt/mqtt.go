package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"os"
	"sync"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"github.com/dnstapir/mqtt-bridge/app/log"
)

type Conf struct {
	Url        string
	CaCert     string
	ClientCert string
	ClientKey  string
	Keylogfile string
	Ctx        context.Context
}

var (
	mqttConnM         *autopaho.ConnectionManager
	mqttCtx           context.Context
	subscriptionsMu   sync.Mutex
	mqttSubscriptions []paho.Subscribe
)

func Init(conf Conf) error {
	/* Parse the URL */
	mqttUrl, err := url.Parse(conf.Url)
	if err != nil {
		return errors.New("Invalid MQTT url")
	}

	log.Info("Url configured: '%s'", conf.Url)

	/* Create a place to remember subscriptions */
	mqttSubscriptions = make([]paho.Subscribe, 0)

	/* Read the CA cert */
	var caCertPool *x509.CertPool = nil
	if conf.CaCert != "" {
		caCertPool = x509.NewCertPool()
		cert, err := os.ReadFile(conf.CaCert)
		if err != nil {
			return errors.New("Error reading CA cert")
		}

		ok := caCertPool.AppendCertsFromPEM([]byte(cert))
		if !ok {
			return errors.New("Error adding CA cert")
		}

		log.Info("CA cert configured: '%s'", conf.CaCert)
	}

	/* Read the client cert and key */
	var clientCert tls.Certificate
	clientCertFound := false
	if conf.ClientCert != "" && conf.ClientKey != "" {
		clientCert, err = tls.LoadX509KeyPair(conf.ClientCert, conf.ClientKey)
		if err != nil {
			return errors.New("Error setting up client certs")
		}

		log.Info("Client cert configured: '%s', '%s'", conf.ClientCert, conf.ClientKey)
		log.Info("TLS cert client ID: '%s'", (*clientCert.Leaf).DNSNames[0])
		clientCertFound = true
	}

	/* Handle keylogfile, if enabled */
	var keylogfile *os.File = nil
	if conf.Keylogfile != "" {
		keylogfile, err = os.OpenFile(conf.Keylogfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return errors.New("Error opening MQTT keylogfile for writing")
		}
	}

	/* Configure paho/autopaho */
	pahoCfg := paho.ClientConfig{
		OnClientError:      onClientError,
		OnServerDisconnect: onServerDisconnect,
	}

	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{mqttUrl},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		OnConnectionUp:                onConnectionUp,
		OnConnectError:                onConnectError,
		ClientConfig:                  pahoCfg,
	}

	/* Configure TLS, if configured */
	if caCertPool != nil {
		tlsCfg := tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS13,
		}

		if clientCertFound {
			tlsCfg.Certificates = []tls.Certificate{clientCert}
		}

		if keylogfile != nil {
			tlsCfg.KeyLogWriter = keylogfile
		}

		cfg.TlsCfg = &tlsCfg
	} else {
		log.Info("No CA set for MQTT TLS, will use unencrypted connection")
	}

	/* Connect to MQTT Broker */
	mqttCtx = conf.Ctx

	mqttConnM, err = autopaho.NewConnection(mqttCtx, cfg)
	if err != nil {
		panic(err)
	}

	err = mqttConnM.AwaitConnection(mqttCtx)
	if err != nil {
		panic(err)
	}

	log.Info("MQTT connection accepted")
	log.Info("Initialized MQTT singleton")
	return nil
}

func Subscribe(topic string, handler func([]byte) (bool, error), subID *int) error {
	mqttConnM.AddOnPublishReceived(func(pr autopaho.PublishReceived) (bool, error) {
		for _, e := range pr.Errs {
			if e != nil {
				log.Error("Error while receiving MQTT message: '%s'", e)
			}
		}

		if pr.AlreadyHandled {
			return true, nil
		}

		/*
		 * Check if the incoming packet is to be handled by the handler
		 * associated with this subscription.
		 */
		var packetSubID int
		packetSubID = *pr.Packet.Properties.SubscriptionIdentifier
		if *subID != packetSubID {
			return false, nil
		}

		return handler(pr.Packet.Payload)
	})

	subscription := paho.SubscribeOptions{
		Topic: topic,
		QoS:   0,
	}

	props := paho.SubscribeProperties{
		SubscriptionIdentifier: subID,
	}

	sub := paho.Subscribe{
		Properties:    &props,
		Subscriptions: []paho.SubscribeOptions{subscription},
	}

	ack, err := mqttConnM.Subscribe(mqttCtx, &sub)
	if err != nil {
		panic(err)
	}
	log.Info("Subscribed: %+v", ack)

	subscriptionsMu.Lock()
	defer subscriptionsMu.Unlock()
	mqttSubscriptions = append(mqttSubscriptions, sub)

	return nil
}

func Publish(topic string, payload []byte) error {
	mqttMsg := paho.Publish{
		QoS:     0,
		Topic:   topic,
		Payload: payload,
		Retain:  false,
	}

	log.Debug("Attempting to publish on topic '%s'...", topic)
	_, err := mqttConnM.Publish(mqttCtx, &mqttMsg)

	if err != nil {
		panic(err)
	}

	log.Debug("Published on topic '%s'!", topic)

	return nil
}

func onClientError(err error) {
	log.Error("client error: %s", err)
}

func onServerDisconnect(d *paho.Disconnect) {
	if d.Properties != nil {
		log.Info("server requested disconnect: %s", d.Properties.ReasonString)
	} else {
		log.Info("server requested disconnect; reason code: %d", d.ReasonCode)
	}
}

func onConnectionUp(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	log.Info("connection up")
	subscriptionsMu.Lock()
	defer subscriptionsMu.Unlock()
	for _, sub := range mqttSubscriptions {
		ack, err := cm.Subscribe(mqttCtx, &sub)
		if err != nil {
			panic(err)
		}
		log.Info("Subscribed via 'onConnectionUp': %+v", ack)
	}
}

func onConnectError(err error) {
	log.Error("error whilst attempting connection: %s", err)
}
