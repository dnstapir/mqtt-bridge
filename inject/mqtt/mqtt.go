package mqtt

import (
    "context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"os"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

type Conf struct {
	Log            shared.LoggerIF
	MqttUrl        string
	MqttCaCert     string
	MqttClientCert string
	MqttClientKey  string
}

type mqttclient struct {
	log          shared.LoggerIF
	autopahoConf autopaho.ClientConfig
    connMan      *autopaho.ConnectionManager
}

const cSCHEME_MQTTS = "mqtts"
const cSCHEME_TLS = "tls"

func Create(conf Conf) (*mqttclient, error) {
	newClient := new(mqttclient)

	if conf.Log == nil {
		return nil, errors.New("nil logger when creating mqtt client")
	}
	newClient.log = conf.Log

	mqttUrl, err := url.Parse(conf.MqttUrl)
	if err != nil {
		return nil, errors.New("invalid mqtt url")
	}

	pahoCfg := paho.ClientConfig{
		OnClientError:      newClient.onClientError,
		OnServerDisconnect: newClient.onServerDisconnect,
	}

	newClient.autopahoConf = autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{mqttUrl},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		OnConnectionUp:                newClient.onConnectionUp,
		OnConnectError:                newClient.onConnectError,
		ClientConfig:                  pahoCfg,
	}

	if mqttUrl.Scheme == cSCHEME_MQTTS || mqttUrl.Scheme == cSCHEME_TLS {
		caCertPool := x509.NewCertPool()
		cert, err := os.ReadFile(conf.MqttCaCert)
		if err != nil {
			return nil, errors.New("error reading mqtt ca cert")
		}
		ok := caCertPool.AppendCertsFromPEM([]byte(cert))
		if !ok {
			return nil, errors.New("error adding ca cert")
		}

		clientKeypair, err := tls.LoadX509KeyPair(conf.MqttClientCert, conf.MqttClientKey)
		if err != nil {
			return nil, errors.New("error setting up client certs")
		}

		tlsCfg := tls.Config{
			RootCAs:      caCertPool,
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{clientKeypair},
		}

		newClient.autopahoConf.TlsCfg = &tlsCfg
	}

	return newClient, nil
}

func (c *mqttclient) Connect() error {
    mqttConnM, err := autopaho.NewConnection(context.Background(), c.autopahoConf)
	if err != nil {
        return err
	}

	err = mqttConnM.AwaitConnection(context.Background())
	if err != nil {
        return err
	}

    c.connMan = mqttConnM

	return nil
}

func (c *mqttclient) Subscribe(topic string) (<-chan []byte, error) {
	return nil, errors.New("mqtt.Subscribe not implemented")
}

func (c *mqttclient) StartPublishing(topic string) (chan<- []byte, error) {
    dataChan := make(chan []byte)

    go func(){
	    for data := range dataChan {
            c.log.Debug("Received data %s", string(data))

	        mqttMsg := paho.Publish{
	        	QoS:     0, // TODO make configurable?
	        	Topic:   topic,
	        	Payload: data,
	        	Retain:  false,
	        }

	        c.log.Debug("Attempting to publish on topic '%s'", topic)
            if c.connMan == nil {
                panic("iiiiiiiiiII")
            }
	        _, err := c.connMan.Publish(context.Background(), &mqttMsg)

	        if err != nil {
                c.log.Error("Failed to publish message on topic '%s'", topic)
	        }

        }
        c.log.Warning("Publishing channel closed for topic '%s'", topic)
    }()

	return dataChan, nil
}

func (c *mqttclient) onClientError(err error) {
	c.log.Error("client error: %s", err)
}

func (c *mqttclient) onServerDisconnect(d *paho.Disconnect) {
	if d.Properties != nil {
		c.log.Error("server requested disconnect: %s", d.Properties.ReasonString)
	} else {
		c.log.Error("server requested disconnect; reason code: %d", d.ReasonCode)
	}
}

func (c *mqttclient) onConnectionUp(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	c.log.Info("connection up")
	//subscriptionsMu.Lock()
	//defer subscriptionsMu.Unlock()
	//for _, sub := range mqttSubscriptions {
	//	ack, err := cm.Subscribe(mqttCtx, &sub)
	//	if err != nil {
	//		panic(err)
	//	}
	//	mc.log.Info("Subscribed via 'onConnectionUp': %+v", ack)
	//}
}

func (c *mqttclient) onConnectError(err error) {
	c.log.Error("error whilst attempting connection: %s", err)
}

//package mqtt
//
//import (
//	"context"
//	"crypto/tls"
//	"crypto/x509"
//	"errors"
//	"net/url"
//	"os"
//	"sync"
//
//	"github.com/eclipse/paho.golang/autopaho"
//	"github.com/eclipse/paho.golang/paho"
//
//	"github.com/dnstapir/mqtt-bridge/app/log"
//)
//
//type Conf struct {
//	Url        string
//	CaCert     string
//	ClientCert string
//	ClientKey  string
//	Keylogfile string
//	Ctx        context.Context
//}
//
//var (
//	mqttConnM         *autopaho.ConnectionManager
//	mqttCtx           context.Context
//	subscriptionsMu   sync.Mutex
//	mqttSubscriptions []paho.Subscribe
//)
//
//func Init(conf Conf) error {
//	/* Parse the URL */
//	mqttUrl, err := url.Parse(conf.Url)
//	if err != nil {
//		return errors.New("Invalid MQTT url")
//	}
//
//	log.Info("Url configured: '%s'", conf.Url)
//
//	/* Create a place to remember subscriptions */
//	mqttSubscriptions = make([]paho.Subscribe, 0)
//
//	/* Read the CA cert */
//	var caCertPool *x509.CertPool = nil
//	if conf.CaCert != "" {
//		caCertPool = x509.NewCertPool()
//		cert, err := os.ReadFile(conf.CaCert)
//		if err != nil {
//			return errors.New("Error reading CA cert")
//		}
//
//		ok := caCertPool.AppendCertsFromPEM([]byte(cert))
//		if !ok {
//			return errors.New("Error adding CA cert")
//		}
//
//		log.Info("CA cert configured: '%s'", conf.CaCert)
//	}
//
//	/* Read the client cert and key */
//	var clientCert tls.Certificate
//	clientCertFound := false
//	if conf.ClientCert != "" && conf.ClientKey != "" {
//		clientCert, err = tls.LoadX509KeyPair(conf.ClientCert, conf.ClientKey)
//		if err != nil {
//			return errors.New("Error setting up client certs")
//		}
//
//		log.Info("Client cert configured: '%s', '%s'", conf.ClientCert, conf.ClientKey)
//		log.Info("TLS cert client ID: '%s'", (*clientCert.Leaf).DNSNames[0])
//		clientCertFound = true
//	}
//
//	/* Handle keylogfile, if enabled */
//	var keylogfile *os.File = nil
//	if conf.Keylogfile != "" {
//		keylogfile, err = os.OpenFile(conf.Keylogfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
//		if err != nil {
//			return errors.New("Error opening MQTT keylogfile for writing")
//		}
//	}
//
//	/* Configure paho/autopaho */
//	pahoCfg := paho.ClientConfig{
//		OnClientError:      onClientError,
//		OnServerDisconnect: onServerDisconnect,
//	}
//
//	cfg := autopaho.ClientConfig{
//		ServerUrls:                    []*url.URL{mqttUrl},
//		KeepAlive:                     20,
//		CleanStartOnInitialConnection: false,
//		SessionExpiryInterval:         60,
//		OnConnectionUp:                onConnectionUp,
//		OnConnectError:                onConnectError,
//		ClientConfig:                  pahoCfg,
//	}
//
//	/* Configure TLS, if configured */
//	if caCertPool != nil {
//		tlsCfg := tls.Config{
//			RootCAs:    caCertPool,
//			MinVersion: tls.VersionTLS13,
//		}
//
//		if clientCertFound {
//			tlsCfg.Certificates = []tls.Certificate{clientCert}
//		}
//
//		if keylogfile != nil {
//			tlsCfg.KeyLogWriter = keylogfile
//		}
//
//		cfg.TlsCfg = &tlsCfg
//	} else {
//		log.Info("No CA set for MQTT TLS, will use unencrypted connection")
//	}
//
//	/* Connect to MQTT Broker */
//	mqttCtx = conf.Ctx
//
//	mqttConnM, err = autopaho.NewConnection(mqttCtx, cfg)
//	if err != nil {
//		panic(err)
//	}
//
//	err = mqttConnM.AwaitConnection(mqttCtx)
//	if err != nil {
//		panic(err)
//	}
//
//	log.Info("MQTT connection accepted")
//	log.Info("Initialized MQTT singleton")
//	return nil
//}
//
//func Subscribe(topic string, handler func([]byte) (bool, error), subID *int) error {
//	mqttConnM.AddOnPublishReceived(func(pr autopaho.PublishReceived) (bool, error) {
//		for _, e := range pr.Errs {
//			if e != nil {
//				log.Error("Error while receiving MQTT message: '%s'", e)
//			}
//		}
//
//		if pr.AlreadyHandled {
//			return true, nil
//		}
//
//		/*
//		 * Check if the incoming packet is to be handled by the handler
//		 * associated with this subscription.
//		 */
//		var packetSubID int
//		packetSubID = *pr.Packet.Properties.SubscriptionIdentifier
//		if *subID != packetSubID {
//			return false, nil
//		}
//
//		return handler(pr.Packet.Payload)
//	})
//
//	subscription := paho.SubscribeOptions{
//		Topic: topic,
//		QoS:   0,
//	}
//
//	props := paho.SubscribeProperties{
//		SubscriptionIdentifier: subID,
//	}
//
//	sub := paho.Subscribe{
//		Properties:    &props,
//		Subscriptions: []paho.SubscribeOptions{subscription},
//	}
//
//	ack, err := mqttConnM.Subscribe(mqttCtx, &sub)
//	if err != nil {
//		panic(err)
//	}
//	log.Info("Subscribed: %+v", ack)
//
//	subscriptionsMu.Lock()
//	defer subscriptionsMu.Unlock()
//	mqttSubscriptions = append(mqttSubscriptions, sub)
//
//	return nil
//}
//
//func Publish(topic string, payload []byte) error {
//	mqttMsg := paho.Publish{
//		QoS:     0,
//		Topic:   topic,
//		Payload: payload,
//		Retain:  false,
//	}
//
//	log.Debug("Attempting to publish on topic '%s'...", topic)
//	_, err := mqttConnM.Publish(mqttCtx, &mqttMsg)
//
//	if err != nil {
//		panic(err)
//	}
//
//	log.Debug("Published on topic '%s'!", topic)
//
//	return nil
//}
//
