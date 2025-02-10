package bridge

import (
    "bytes"
    "context"
	"crypto/tls"
	"crypto/x509"
    "errors"
	"net/url"
    "os"
    "strings"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/nats-io/nats.go"
    "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/dnstapir/mqtt-sender/app/log"
)

type bridgeOpt func(*tapirBridge) error

type tapirBridge struct {
    direction        string
	mqttUrl          *url.URL
	caCert           *x509.CertPool
	clientCert       tls.Certificate
	keylogfile       *os.File
    enableKeylogfile bool
    natsUrl          string
    topic            string
    subject          string
    queue            string
	dataKey          jwk.Key
    schema           *jsonschema.Schema

    natsMsgCh  chan *nats.Msg
    natsConn   *nats.Conn
    mqttConnM  *autopaho.ConnectionManager
	ctx        context.Context
    clientId   string
}

func Create(direction string, options ...bridgeOpt) (*tapirBridge, error) {
    if direction != "up" && direction != "down" {
        return nil, errors.New("Bridge must be either 'up' or 'down'")
    }

	newBridge := new(tapirBridge)
    newBridge.direction = direction

	for _, opt := range options {
		err := opt(newBridge)
		if err != nil {
			panic(err)
		}
	}

    log.Info("Done setting up %sbound bridge between %s and %s",
        newBridge.direction,
        newBridge.topic,
        newBridge.subject,
    )

    return newBridge, nil
}

func (tb *tapirBridge) Start() error {
    if tb.isUpbound() {
        return tb.startUpbound()
    } else {
        return tb.startDownbound()
    }

    return nil
}

func (tb *tapirBridge) isUpbound() bool {
    if tb.direction == "up" {
        return true
    }

    return false
}

func (tb *tapirBridge) startUpbound() error {
	tb.ctx = context.Background() // TODO get this context thing right

    /* For sanity, compare "kid" of data key and first DNSName SAN of tls cert */
	cidDataKey := tb.dataKey.KeyID()
	cidTlsCert := (*tb.clientCert.Leaf).DNSNames[0]
	log.Info("JWK client ID: '%s', TLS cert client ID: '%s'", cidDataKey, cidTlsCert)

	tb.clientId = cidDataKey

	tlsCfg := tls.Config{
		RootCAs:      tb.caCert,
		Certificates: []tls.Certificate{tb.clientCert},
		MinVersion:   tls.VersionTLS13,
	}

	if tb.enableKeylogfile {
		tlsCfg.KeyLogWriter = tb.keylogfile
	}

	cfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{tb.mqttUrl},
		TlsCfg:     &tlsCfg,
		KeepAlive:  20, // Keepalive message should be sent every 20 seconds
		// CleanStartOnInitialConnection defaults to false. Setting this to true will clear the session on the first connection.
		CleanStartOnInitialConnection: false,
		// SessionExpiryInterval - Seconds that a session will survive after disconnection.
		// It is important to set this because otherwise, any queued messages will be lost if the connection drops and
		// the server will not queue messages while it is down. The specific setting will depend upon your needs
		// (60 = 1 minute, 3600 = 1 hour, 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			_, err := cm.Subscribe(tb.ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: tb.topic, QoS: 0},
				},
			})
            if err != nil {
				log.Error("subscribe: failed to subscribe (%s). Probably due to connection drop so will retry", err)
				return // likely connection has dropped
			}
			log.Info("Subsribed to '%s'", tb.topic)
		},
		OnConnectError: func(err error) { log.Error("error whilst attempting connection: %s", err) },

		// eclipse/paho.golang/paho provides base mqtt functionality, the below config will be passed in for each connection
		ClientConfig: paho.ClientConfig{
			// If you are using QOS 1/2, then it's important to specify a client id (which must be unique)
			ClientID:      tb.clientId,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
                    panic("got message!")
					return true, nil
				}},
			OnClientError: func(err error) { log.Error("client error: %s", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					log.Info("server requested disconnect: %s", d.Properties.ReasonString)
				} else {
					log.Info("server requested disconnect; reason code: %d", d.ReasonCode)
				}
			},
		},
	}

	cm, err := autopaho.NewConnection(tb.ctx, cfg) // starts process; will reconnect until context cancelled
	if err != nil {
		panic(err)
	}
	tb.mqttConnM = cm

	// Wait for the connection to come up
	err = tb.mqttConnM.AwaitConnection(tb.ctx)
	if err != nil {
		panic(err)
	}
    log.Info("upbound started")

    return nil
}

func (tb *tapirBridge) startDownbound() error {
    err := tb.startDownboundNats()
    if err != nil {
        panic(err)
    }

    err = tb.startDownboundMqtt()
    if err != nil {
        panic(err)
    }

	go tb.loopDownbound()

    return nil
}

func (tb *tapirBridge) loopDownbound() {
	log.Info("Downbound loop started")

    for msg := range tb.natsMsgCh {
        log.Debug("Got message '%s'", string(msg.Data))

        ok, err := tb.validateWithSchema(msg.Data)
        if err != nil {
            panic(err)
        }

        if ok {
	        // Do the signing sauce
            payload, err := jws.Sign(msg.Data, jws.WithJSON(), jws.WithKey(tb.dataKey.Algorithm(), tb.dataKey))
	        if err != nil {
	        	panic(err)
	        }

            err = tb.publishObservation(payload)
	        if err != nil {
	        	panic(err)
	        }
        } else {
            log.Error("Malformed data '%s', discarding...", string(msg.Data))
        }

        msg.Ack()
    }

    tb.natsConn.Close()
    close(tb.natsMsgCh) // TODO close these in the right order

	log.Info("Downbound loop stopped")
}

func (tb *tapirBridge) publishObservation(payload []byte) error {
	if tb.mqttConnM == nil {
		panic("MQTT not initialized")
	}

	// Wait for the connection to come up
	err := tb.mqttConnM.AwaitConnection(tb.ctx)
	if err != nil {
		panic(err)
	}

	mqttMsg := paho.Publish{
		QoS:     0,
		Topic:   tb.topic,
		Payload: payload,
		Retain:  false,
	}

	log.Debug("Attempting to publish on topic '%s'...", tb.topic)
	_, err = tb.mqttConnM.Publish(tb.ctx, &mqttMsg)

	if err != nil {
		panic(err)
	}
	log.Debug("Published on topic '%s'!", tb.topic)

	return nil
}

func (tb *tapirBridge) startDownboundNats() error {
    nc, err := nats.Connect(tb.natsUrl)
    if err != nil {
        return err
    }

    msgCh := make(chan *nats.Msg)
    _, err = nc.ChanQueueSubscribe(tb.subject, tb.queue, msgCh)
    if err != nil {
        return err
    }

    tb.natsMsgCh = msgCh
    tb.natsConn = nc
    return nil
}

func (tb *tapirBridge) startDownboundMqtt() error {
	tb.ctx = context.Background() // TODO get this context thing right

    /* For sanity, compare "kid" of data key and first DNSName SAN of tls cert */
	cidDataKey := tb.dataKey.KeyID()
	cidTlsCert := (*tb.clientCert.Leaf).DNSNames[0]
	log.Info("JWK client ID: '%s', TLS cert client ID: '%s'", cidDataKey, cidTlsCert)

	tb.clientId = cidDataKey

	tlsCfg := tls.Config{
		RootCAs:      tb.caCert,
		Certificates: []tls.Certificate{tb.clientCert},
		MinVersion:   tls.VersionTLS13,
	}

	if tb.enableKeylogfile {
		tlsCfg.KeyLogWriter = tb.keylogfile
	}

	cfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{tb.mqttUrl},
		TlsCfg:     &tlsCfg,
		KeepAlive:  20, // Keepalive message should be sent every 20 seconds
		// CleanStartOnInitialConnection defaults to false. Setting this to true will clear the session on the first connection.
		CleanStartOnInitialConnection: false,
		// SessionExpiryInterval - Seconds that a session will survive after disconnection.
		// It is important to set this because otherwise, any queued messages will be lost if the connection drops and
		// the server will not queue messages while it is down. The specific setting will depend upon your needs
		// (60 = 1 minute, 3600 = 1 hour, 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Debug("mqtt connection up")
		},
		OnConnectError: func(err error) { log.Error("error whilst attempting connection: %s", err) },

		// eclipse/paho.golang/paho provides base mqtt functionality, the below config will be passed in for each connection
		ClientConfig: paho.ClientConfig{
			// If you are using QOS 1/2, then it's important to specify a client id (which must be unique)
			ClientID:      tb.clientId,
			OnClientError: func(err error) { log.Error("client error: %s", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					log.Info("server requested disconnect: %s", d.Properties.ReasonString)
				} else {
					log.Info("server requested disconnect; reason code: %d", d.ReasonCode)
				}
			},
		},
	}

	cm, err := autopaho.NewConnection(tb.ctx, cfg) // starts process; will reconnect until context cancelled
	if err != nil {
		panic(err)
	}
	tb.mqttConnM = cm

    return nil
}

func (tb *tapirBridge) validateWithSchema(data []byte) (bool, error) {
    if tb.schema == nil {
        log.Error("No schema set for subject '%s'", tb.subject)
        return false, errors.New("Cannot validate JSON without schema")
    }

    dataReader := bytes.NewReader(data)
    obj, err := jsonschema.UnmarshalJSON(dataReader)
    if err != nil {
        return false, errors.New("Error unmarshalling byte stream into JSON object")
    }

    err = tb.schema.Validate(obj)
    if err != nil {
        // TODO handle error more explicitly, make sure is "ValidationError"
        log.Debug("Validation error '%s'", err)
        return false, nil
    }

    return true, nil
}

/*
 ******************************************************************************
 ************    HERE BE CONFIGURATION PARAMETERS     *************************
 ******************************************************************************
 */

func MqttUrl(urlRaw string) bridgeOpt {
	fPtr := func(tb *tapirBridge) error {
		u, err := url.Parse(urlRaw)
		if err != nil {
			return errors.New("Invalid MQTT url")
		}
		tb.mqttUrl = u

		log.Info("Url configured: '%s'", u)
		return nil
	}

	return fPtr
}

func CaCert(filename string) bridgeOpt {
	fPtr := func(tb *tapirBridge) error {
		caCertPool := x509.NewCertPool()
		cert, err := os.ReadFile(filename)
		if err != nil {
			return errors.New("Error reading CA cert")
		}

		ok := caCertPool.AppendCertsFromPEM([]byte(cert))

		if !ok {
			return errors.New("Error adding CA cert")
		}

		tb.caCert = caCertPool

		log.Info("CA cert configured: '%s'", filename)
		return nil
	}

	return fPtr
}

func ClientCert(certFilename, keyFilename string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
		clientCert, err := tls.LoadX509KeyPair(certFilename, keyFilename)

		if err != nil {
			return errors.New("Error setting up client certs")
		}

		/*
		 * Get parsed client cert, not needed after go 1.23
		 * When we have the parsed version, we can extract the CN
		 */
		rawCert := clientCert.Certificate[0]
		clientCert.Leaf, err = x509.ParseCertificate(rawCert)
		if err != nil {
			return errors.New("Error parsing client certificate")
		}

		tb.clientCert = clientCert

		log.Info("Client cert configured: '%s', '%s'", certFilename, keyFilename)
		return nil
	}

	return fptr
}

func TlsKeylogfile(enable bool, filename string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
		if !enable {
			if filename != "" {
				log.Warning("Keylogfile disabled, but outfile set. Ignoring...")
			}
			return nil
		}

		if filename == "" {
			log.Warning("Keylogfile enabled, but no outfile set. Ignoring...")
			return nil
		}

		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return errors.New("Error opening keylogfile for writing")
		}

		tb.keylogfile = f
        tb.enableKeylogfile = enable

		log.Warning("Logging TLS keys to '%s'", filename)
		return nil
	}

	return fptr
}

func NatsUrl(urlRaw string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
		u, err := url.Parse(urlRaw)
		if err != nil {
			return errors.New("Invalid NATS url")
		}
		tb.natsUrl = u.String()

		log.Info("NATS Url configured: '%s'", u)
		return nil
	}

	return fptr
}

func Topic(topic string) bridgeOpt {
	fPtr := func(tb *tapirBridge) error {
        /* We don't support wildcards when we are publishing to MQTT (downbound) */
        if !tb.isUpbound() && strings.ContainsAny(topic, "#") {
            return errors.New("Wildcard topics not allowed for downbound bridges")
        }

		tb.topic = topic

		log.Info("Topic configured: '%s'", topic)
		return nil
	}

	return fPtr
}

func Subject(subject string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
        /* We don't support wildcards when we are publishing to NATS (upbound) */
        if tb.isUpbound() && strings.ContainsAny(subject, ">*") {
            return errors.New("Wildcard topics not allowed for downbound bridges")
        }
		tb.subject = subject

		log.Info("NATS subject configured: '%s'", subject)
		return nil
	}

	return fptr
}

func Queue(queue string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
        // TODO any name restrictions on queue?
		tb.queue = queue

		log.Info("NATS queue configured: '%s'", queue)
		return nil
	}

	return fptr
}

func Schema(filename string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
        if filename == "" {
            log.Warning("No JSON schema configured!")
            return nil
        }
        // TODO support fetching from URL?

        c := jsonschema.NewCompiler()
        sch, err := c.Compile(filename)
        if err != nil {
            return err
        }

        tb.schema = sch

		return nil
	}

	return fptr
}

func DataKey(filename string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
		keyFile, err := os.ReadFile(filename)
		if err != nil {
			return errors.New("Error reading signing key file")
		}

		keyParsed, err := jwk.ParseKey(keyFile)
		if err != nil {
			return errors.New("Error parsing signing key file")
		}

        isPrivate, err := jwk.IsPrivateKey(keyParsed)
        if err != nil {
			return errors.New("Error checking assymatric key")
        }

        /*
         * Use private (signing) key for downbound bridges and public
         * (validating) key for upbound bridges
         */
        if tb.isUpbound() && isPrivate {
            return errors.New("Upbound bridges must use a public (validating) key")
        } else if !tb.isUpbound() && !isPrivate {
            return errors.New("Downbound bridges must use a private (signing) key")
        }

		tb.dataKey = keyParsed

		log.Info("Data key configured: '%s'", filename)
		return nil
	}

	return fptr
}
