package mqtt

import (
    "bytes"
    "context"
    "crypto/tls"
    "crypto/x509"
    "encoding/json"
    "errors"
    "net/url"
    "os"
    "strconv"
    "time"

    "github.com/dnstapir/tapir"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

    "tapir-core-mqtt-sender/app/log"
)

type Mqtt struct {
    url *url.URL
    topic string
    caCert *x509.CertPool
    clientCert tls.Certificate
    keylogfile *os.File
    enableKeylogfile bool
    signingKey jwk.Key
    clientId string
    ctx context.Context
    cm *autopaho.ConnectionManager
}

func Create(options ...func(*Mqtt) error) (*Mqtt, error) {
    m := new(Mqtt)

    for _, opt := range options {
        err := opt(m)
        if err != nil {
            panic(err)
        }
    }

    log.Debug("Options set succesfully")

    m.ctx = context.Background()

    cidSigningKey := m.signingKey.KeyID()
    cidTlsCert := (*m.clientCert.Leaf).Subject.CommonName
    log.Debug("JWK client ID: '%s', TLS cert client ID: '%s'", cidSigningKey, cidTlsCert)
    if cidSigningKey != cidTlsCert {
        panic(errors.New("Name mismatch for signing key and TLS cert"))
    }
    m.clientId = cidSigningKey

    tlsCfg := tls.Config{
			RootCAs:      m.caCert,
			Certificates: []tls.Certificate{m.clientCert},
			MinVersion:   tls.VersionTLS13,
	}

    if m.enableKeylogfile {
        tlsCfg.KeyLogWriter = m.keylogfile
    }

	cfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{m.url},
		TlsCfg: &tlsCfg,
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
			ClientID: m.clientId,
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

    cm, err := autopaho.NewConnection(m.ctx, cfg) // starts process; will reconnect until context cancelled
	if err != nil {
		panic(err)
	}
    m.cm = cm

    return m, nil
}

func Url(urlRaw string) func(*Mqtt) error {
    fPtr := func(m *Mqtt) error {
        u, err := url.Parse(urlRaw)
        if err != nil {
            return errors.New("Invalid url")
        }
        m.url = u

        log.Info("Url configured: '%s'", u)
        return nil
    }

    return fPtr
}

func Topic(topic string) func(*Mqtt) error {
    fPtr := func(m *Mqtt) error {
        m.topic = topic

        log.Info("Topic configured: '%s'", topic)
        return nil
    }

    return fPtr
}

func CaCert(filename string) func(*Mqtt) error {
    fPtr := func(m *Mqtt) error {
        caCertPool := x509.NewCertPool()
        cert, err := os.ReadFile(filename)
        if err != nil {
            return errors.New("Error reading CA cert")
        }

        ok := caCertPool.AppendCertsFromPEM([]byte(cert))

        if !ok {
            return errors.New("Error adding CA cert")
        }

        m.caCert = caCertPool

        log.Info("CA cert configured: '%s'", filename)
        return nil
    }

    return fPtr
}

func ClientCert(certFilename, keyFilename string) func(*Mqtt) error {
    fptr := func(m *Mqtt) error {
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

        m.clientCert = clientCert

        log.Info("Client cert configured: '%s', '%s'", certFilename, keyFilename)
        return nil
    }

    return fptr
}

func TlsKeylogfile(enable bool, filename string) func(*Mqtt) error {
    fptr := func(m *Mqtt) error {
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

        m.enableKeylogfile = enable
        m.keylogfile = f

        log.Warning("Logging TLS keys to '%s'", filename)
        return nil
    }

    return fptr
}

func SigningKey(filename string) func(*Mqtt) error {
    fptr := func(m *Mqtt) error {
        keyFile, err := os.ReadFile(filename)
        if err != nil {
            return errors.New("Error reading signing key file")
        }

        keyParsed, err := jwk.ParseKey(keyFile)
        if err != nil {
            return errors.New("Error parsing signing key file")
        }

        m.signingKey = keyParsed

        log.Info("Signing key configured: '%s'", filename)
        return nil
    }

    return fptr
}

func (m *Mqtt) Publish(domain string, data []byte) error {
    if m.cm == nil {
        panic("MQTT not initialized")
    }
    log.Debug("attempting to publish '%s'", data)// Not really though

	// Wait for the connection to come up
    err := m.cm.AwaitConnection(m.ctx)
    if err != nil {
		panic(err)
	}

    // The actual Tapir payload where the magic lives
    obs := tapir.TapirMsg{
        SrcName:"dns-tapir",
        //Creator:,
        MsgType: "observation",
        ListType:"doubtlist",
        //Msg:,
        TimeStamp:time.Now(),
        //TimeStr:,
    }

    tags, err := strconv.ParseUint(string(data), 10, 32)
    if err != nil {
        panic(err)
    }
    tags32 := uint32(tags)

    evilDomain := tapir.Domain{
        Name: domain,
        TimeAdded: time.Now(),
        TTL: 3600,
        TagMask: tapir.TagMask(tags32),
    }
    obs.Added = []tapir.Domain{evilDomain}

    // Create something to JSON encode the tapir message in
	buf := new(bytes.Buffer)
	jenc := json.NewEncoder(buf)

    err = jenc.Encode(obs)
    if err != nil {
        panic(err)
    }

    // Do the signing sauce
    payload, err := jws.Sign(buf.Bytes(), jws.WithJSON(), jws.WithKey(jwa.ES256, m.signingKey))
    if err != nil {
        panic(err)
    }

    msg := paho.Publish{
        QoS: 2,
        Topic: m.topic,
        Payload: payload,
        Retain: false,
    }

    _, err = m.cm.Publish(m.ctx, &msg)

    if err != nil {
		panic(err)
	}

    log.Debug("Published on topic '%s'!", m.topic)

    return nil
}
