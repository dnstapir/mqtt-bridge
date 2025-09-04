package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

const c_MQTT_TIMEOUT = 30

type Conf struct {
	Log            shared.LoggerIF
	MqttUrl        string
	MqttCaCert     string
	MqttClientCert string
	MqttClientKey  string
}

type mqttclient struct {
	log               shared.LoggerIF
	autopahoConf      autopaho.ClientConfig
	connMan           *autopaho.ConnectionManager
	subscriptionsMu   sync.Mutex
	subscriptions     subscriptionsMu
	subscriptionOutCh chan []byte
	done              chan struct{}
	connectionOk      connectionStatusMu
}

type subscriptionsMu struct {
	sync.RWMutex
	subs []paho.SubscribeOptions
}

type connectionStatusMu struct {
	sync.RWMutex
	ok bool
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

	newClient.subscriptionOutCh = make(chan []byte, 1024)
	newClient.done = make(chan struct{})
	newClient.subscriptions.Lock()
	newClient.subscriptions.subs = make([]paho.SubscribeOptions, 0)
	newClient.subscriptions.Unlock()

	pahoCfg := paho.ClientConfig{
		OnClientError:      newClient.onClientError,
		OnServerDisconnect: newClient.onServerDisconnect,
		OnPublishReceived: []func(paho.PublishReceived) (bool, error){
			newClient.subscriptionCb,
		},
	}

	newClient.autopahoConf = autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{mqttUrl},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         500,
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

	c.connMan = mqttConnM

	ctx, cancel := context.WithTimeout(context.Background(), c_MQTT_TIMEOUT*time.Second)
	err = mqttConnM.AwaitConnection(ctx)
    cancel()
	if err != nil {
		return err
	}

	c.connectionOk.Lock()
	defer c.connectionOk.Unlock()
	c.connectionOk.ok = true

	return nil
}

func (c *mqttclient) subscriptionCb(pr paho.PublishReceived) (bool, error) {
	c.log.Debug("Received %d bytes on topic '%s'", len(pr.Packet.Payload), pr.Packet.Topic)
	for _, e := range pr.Errs {
		if e != nil {
			c.log.Error("Error while receiving MQTT message: '%s'", e)
		}
	}

	if pr.AlreadyHandled {
		return true, nil
	}

	go func() {
		select {
		case c.subscriptionOutCh <- pr.Packet.Payload:
			c.log.Debug("Successfully handled packet on topic '%s'", pr.Packet.Topic)
		case <-c.done:
			c.log.Warning("Shutdown signaled, dropping incoming mqtt packet")
			return
		}
	}()

	return true, nil
}

func (c *mqttclient) Subscribe(topic string) (<-chan []byte, error) {
	subscription := paho.SubscribeOptions{
		Topic: topic,
		QoS:   0,
	}

	c.subscriptions.Lock()
	c.subscriptions.subs = append(c.subscriptions.subs, subscription)
	c.subscriptions.Unlock()

	c.log.Info("Topic '%s' added to pending subscriptions", topic)

	c.connectionOk.RLock()
	connectionOk := c.connectionOk.ok
	c.connectionOk.RUnlock()

	if connectionOk {
		c.log.Info("Will attempt to subscribe to '%s'", topic)
		sub := paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{subscription},
		}
		ctx, cancel := context.WithTimeout(context.Background(), c_MQTT_TIMEOUT*time.Second)
		_, err := c.connMan.Subscribe(ctx, &sub)
		cancel()
		if err != nil {
			// Connection was up, but we couldn't reconnect
			c.log.Warning("Failed to subscribe to topic '%s': %s", err)
			c.log.Info("Will attempt to subscribe again once connection is stable")
		}
	}

	return c.subscriptionOutCh, nil
}

func (c *mqttclient) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), c_MQTT_TIMEOUT*time.Second)
	defer cancel()

	c.subscriptions.RLock()
	subsCopy := make([]paho.SubscribeOptions, len(c.subscriptions.subs))
	copy(subsCopy, c.subscriptions.subs)
	c.subscriptions.RUnlock()

	unsub := new(paho.Unsubscribe)

	for _, s := range subsCopy {
		unsub.Topics = append(unsub.Topics, s.Topic)
	}

	if len(unsub.Topics) > 0 && c.connMan != nil {
		_, err := c.connMan.Unsubscribe(ctx, unsub)
		if err != nil {
			c.log.Error("Unubsribe failed: %s", err)
		}
	}

	c.subscriptions.Lock()
	c.subscriptions.subs = nil
	c.subscriptions.Unlock()

	close(c.done)
	time.Sleep(10 * time.Millisecond)
	close(c.subscriptionOutCh)
}

func (c *mqttclient) StartPublishing(topic string) (chan<- []byte, error) {
	dataChan := make(chan []byte)

	if c.connMan == nil {
		return nil, errors.New("mqtt client must connect first")
	}

	go func() {
		for data := range dataChan {
			var err error

			mqttMsg := paho.Publish{
				QoS:     0, // TODO make configurable?
				Topic:   topic,
				Payload: data,
				Retain:  false,
			}

			c.log.Debug("Attempting to publish on topic '%s'", topic)

			ctx, cancel := context.WithTimeout(context.Background(), c_MQTT_TIMEOUT*time.Second)
			err = c.connMan.AwaitConnection(ctx)
			if err != nil {
				c.log.Error("Error while awaiting MQTT connection")
				cancel()
				continue
			}

			_, err = c.connMan.Publish(ctx, &mqttMsg)

			if err != nil {
				c.log.Error("Error '%s' while publishing on topic '%s'", err, topic)
			} else {
				c.log.Debug("Successfully published %d bytes on MQTT topic '%s'", len(mqttMsg.Payload), topic)
			}
			cancel()
		}

		c.log.Warning("Publishing channel closed for topic '%s'", topic)
	}()

	return dataChan, nil
}

func (c *mqttclient) CheckConnection() bool {
	var ok bool

	c.connectionOk.RLock()
	ok = c.connectionOk.ok
	c.connectionOk.RUnlock()

	return ok
}

func (c *mqttclient) onClientError(err error) {
	c.log.Info("Client error: %s", err)

	c.connectionOk.Lock()
	c.connectionOk.ok = false
	c.connectionOk.Unlock()
}

func (c *mqttclient) onServerDisconnect(d *paho.Disconnect) {
	c.log.Info("Server disconnected!")
	c.connectionOk.Lock()
	c.connectionOk.ok = false
	c.connectionOk.Unlock()

	if d.Properties != nil {
		c.log.Error("server requested disconnect: %s", d.Properties.ReasonString)
	} else {
		c.log.Error("server requested disconnect; reason code: %d", d.ReasonCode)
	}
}

func (c *mqttclient) onConnectionUp(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	c.log.Info("connection came up, will subscribe")

	c.subscriptions.RLock()
	subsCopy := make([]paho.SubscribeOptions, len(c.subscriptions.subs))
	copy(subsCopy, c.subscriptions.subs)
	c.subscriptions.RUnlock()

	if len(subsCopy) != 0 {
		sub := paho.Subscribe{
			Subscriptions: subsCopy,
		}

		ctx, cancel := context.WithTimeout(context.Background(), c_MQTT_TIMEOUT*time.Second)
		_, err := c.connMan.Subscribe(ctx, &sub)
		cancel()
		if err != nil {
			c.log.Error("Failed to subscribe on connection-up: %s", err)
		}
		c.log.Info("Subscribed to %d topics when connection came up", len(subsCopy))
	}

	c.connectionOk.Lock()
	c.connectionOk.ok = true
	c.connectionOk.Unlock()

	c.log.Info("connection up and ready for use!")
}

func (c *mqttclient) onConnectError(err error) {
	c.log.Error("error whilst attempting connection: %s", err)
	c.connectionOk.Lock()
	c.connectionOk.ok = false
	c.connectionOk.Unlock()
}
