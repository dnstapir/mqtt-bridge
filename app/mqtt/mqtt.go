package mqtt

import (
    "bytes"
    "context"
    "crypto/tls"
    "crypto/x509"
    "encoding/json"
    "encoding/pem"
    "net/url"
    "os"
    "time"

    "github.com/dnstapir/tapir"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"

    "tapir-core-mqtt-sender/app/log"
)

var ctx = context.Background()
var cm *autopaho.ConnectionManager

func Initialize() error {
	// We will connect to the Eclipse test server (note that you may see messages that other users publish)
    u, err := url.Parse("tls://mqtt.dev.dnstapir.se:8883")
	if err != nil {
		panic(err)
	}

    // Get the server cert
	caCertPool := x509.NewCertPool()
	cert, err := os.ReadFile("/path/to/ca.crt")
    if err != nil {
        panic(err)
    }
	ok := caCertPool.AppendCertsFromPEM([]byte(cert))
    if !ok {
        panic("Error adding CA cert")
    }

    // And the client cert+key
	clientCert, err := tls.LoadX509KeyPair("/path/to/tls.crt", "/path/to/tls.key")
    if err != nil {
        panic(err)
    }

    //f, err := os.OpenFile("/tmp/keys", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
    if err != nil {
      panic(err)
    }


	cfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{u},
		TlsCfg: &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{clientCert},
			MinVersion:   tls.VersionTLS13,
            //KeyLogWriter: f, // TODO remove!
		},
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
			ClientID: "admiring-albattani.dev.dnstapir.se",
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

	cm, err = autopaho.NewConnection(ctx, cfg) // starts process; will reconnect until context cancelled
	if err != nil {
		panic(err)
	}

    return nil
}

func Publish(data []byte) error {
    if cm == nil {
        panic("MQTT not initialized")
    }
    log.Debug("attempting to publish '%s'", data)

	// Wait for the connection to come up
    err := cm.AwaitConnection(ctx)
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
    evilDomain := tapir.Domain{
        Name: "moahahahahahaha.evil.domain.example.com.",
        TimeAdded: time.Now(),
        TTL: 3600,
        TagMask: 256,
    }
    obs.Added = []tapir.Domain{evilDomain}

    // Create something to JSON encode the tapir message in
	buf := new(bytes.Buffer)
	jenc := json.NewEncoder(buf)

    err = jenc.Encode(obs)
    if err != nil {
        panic(err)
    }

    // Fetch the signing key
    keyFile, err := os.ReadFile("/path/to/data.key")
    if err != nil {
        panic(err)
    }
	pemBlock, _ := pem.Decode(keyFile)
	//if pemBlock == nil || pemBlock.Type != "EC PRIVATE KEY" {
    //    panic("Error decoding PEM block")
    //}
	signingKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
        panic(err)
	}

    // Do the signing sauce
    payload, err := jws.Sign(buf.Bytes(), jws.WithJSON(), jws.WithKey(jwa.ES256, signingKey))
    if err != nil {
        panic(err)
    }

    msg := paho.Publish{
        QoS: 2,
        Topic: "observations/down/tapir-pop",
        Payload: payload,
        Retain: false,
    }

    _, err = cm.Publish(ctx, &msg)

    if err != nil {
		panic(err)
	}

    log.Debug("Published!")

    return nil
}
