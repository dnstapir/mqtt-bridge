package mqtt

import (
    "context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"os"
	"sync"

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
    subscriptionsMu   sync.Mutex
	mqttSubscriptions []paho.Subscribe
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

    newClient.mqttSubscriptions = make([]paho.Subscribe, 0)

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
	byteCh := make(chan []byte)
    callback := func(pr autopaho.PublishReceived) (bool, error) {
		for _, e := range pr.Errs {
			if e != nil {
				c.log.Error("Error while receiving MQTT message: '%s'", e)
                panic(e)
			}
		}

		if pr.AlreadyHandled {
			return true, nil
		}

        byteCh <- pr.Packet.Payload

		return true, nil
	}

    c.connMan.AddOnPublishReceived(callback)

    subscription := paho.SubscribeOptions{
		Topic: topic,
		QoS:   0,
	}

	sub := paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{subscription},
	}

	ack, err := c.connMan.Subscribe(context.Background(), &sub)
	if err != nil {
		panic(err)
	}
	c.log.Info("Subscribed: %+v", ack)

	c.subscriptionsMu.Lock()
	defer c.subscriptionsMu.Unlock()
	c.mqttSubscriptions = append(c.mqttSubscriptions, sub)

	return byteCh, nil
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
    panic(err)
}

func (c *mqttclient) onServerDisconnect(d *paho.Disconnect) {
	if d.Properties != nil {
		c.log.Error("server requested disconnect: %s", d.Properties.ReasonString)
	} else {
		c.log.Error("server requested disconnect; reason code: %d", d.ReasonCode)
	}
    panic("disconnect")
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
    panic(err)
}
