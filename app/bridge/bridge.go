package bridge

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/nats-io/nats.go"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/dnstapir/mqtt-bridge/app/log"
)

type bridgeOpt func(*tapirBridge) error

type tapirBridge struct {
	direction         string
	keylogfileNodeman *os.File
	enableKeylogfile  bool
	natsUrl           string
	topic             string
    bridgeID          int
	subject           string
	queue             string
	dataKey           jwk.Key
	schema            *jsonschema.Schema
	nodemanApi        *url.URL
    publish           func(string, []byte) error

	natsMsgCh  chan *nats.Msg
	natsConn   *nats.Conn
	ctx        context.Context
	httpClient http.Client

	validationKeyCache *lru.Cache[string, jwk.Key]
}

const validateKeyCacheSize = 1000

func Create(direction string, id int, options ...bridgeOpt) (*tapirBridge, error) {
	if direction != "up" && direction != "down" {
		return nil, errors.New("Bridge must be either 'up' or 'down'")
	}

	newBridge := new(tapirBridge)
	newBridge.direction = direction
    newBridge.bridgeID = id

	for _, opt := range options {
		err := opt(newBridge)
		if err != nil {
			panic(err)
		}
	}

	if newBridge.isUpbound() {
		if newBridge.dataKey == nil && newBridge.nodemanApi == nil {
			return nil, errors.New("Upbound bridge without validation keys")
		} else if newBridge.dataKey != nil && newBridge.nodemanApi != nil {
			log.Warning("Using both remote and local validation keys")
		}

		var err error
		newBridge.validationKeyCache, err = lru.New[string, jwk.Key](validateKeyCacheSize)
		if err != nil {
			return nil, err
		}
	}

	log.Info("Done setting up %sbound bridge between %s and %s",
		newBridge.direction,
		newBridge.topic,
		newBridge.subject,
	)

	return newBridge, nil
}

func (tb *tapirBridge) Topic() string {
    return tb.topic
}

func (tb *tapirBridge) BridgeID() *int {
    return &tb.bridgeID
}

func (tb *tapirBridge) IncomingPktHandler(payload []byte) (bool, error) {
	jwsMsg, err := jws.Parse(payload, jws.WithJSON())
	if err != nil {
		log.Error("Malformed JWS message '%s'. Discarding...", string(payload))
		return true, err
	}

	sigs := jwsMsg.Signatures()
	if len(sigs) > 1 {
		log.Warning("JWS message contains multiple signatures. Only one will be used")
	} else if len(sigs) == 0 {
		log.Error("JWS message contained no signatures. Discarding...")
		return true, errors.New("JWS message contained no signatures")
	}

	jwsKid := sigs[0].ProtectedHeaders().KeyID()
	if jwsKid == "" {
		log.Error("Incoming JWS had no \"kid\" set. Discarding...")
		return true, errors.New("Incoming JWS had no \"kid\" set")
	}
	jwsAlg := sigs[0].ProtectedHeaders().Algorithm()
	jwsKey, ok := tb.validationKeyCache.Get(jwsKid)

	if !ok {
		req, err := http.NewRequest("GET",
			tb.nodemanApi.JoinPath(fmt.Sprintf("/node/%s/public_key", jwsKid)).String(),
			nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("accept", "application/json")

		rsp, err := tb.httpClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer rsp.Body.Close()

		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			panic(err)
		}

		newJwk, err := jwk.ParseKey(body)
		if err != nil {
			panic(err)
		}

		if newJwk.KeyID() != jwsKid {
			log.Warning("Mismatch between Key IDs in JWS '%s' and '%s'", newJwk.KeyID(), jwsKid)
		}

		tb.validationKeyCache.Add(jwsKid, newJwk)
		jwsKey = newJwk

		log.Info("Adding new key '%s' to cache", newJwk.KeyID())
	}

	data, err := jws.Verify(payload, jws.WithJSON(), jws.WithKey(jwsAlg, jwsKey))
	if err != nil {
		log.Error("Failed to verify signature on message. Discarding...")
		return true, err
	}
	log.Debug("Message signature was successfully validated! Used key '%s'", jwsKid)

	ok, err = tb.validateWithSchema(data)
	if err != nil {
		log.Error("Error validating message against schema. Discarding...")
		return true, err
	}

	if !ok {
		log.Error("Malformed data on message. Discarding...")
		return true, err
	}
	log.Debug("Message conforms to schema")

	err = tb.natsConn.Publish(tb.subject, data)
	if err != nil {
		panic(err)
	}

	return true, nil
}

func (tb *tapirBridge) SetPublishMethod(f func(string, []byte) error) {
    tb.publish = f
}

func (tb *tapirBridge) Start() error {
	if tb.isUpbound() {
		return tb.startUpbound()
	} else {
		return tb.startDownbound()
	}
}

func (tb *tapirBridge) isUpbound() bool {
	if tb.direction == "up" {
		return true
	}

	return false
}

func (tb *tapirBridge) startUpbound() error {
	// NATS startup, TODO move somewere else?
	nc, err := nats.Connect(tb.natsUrl)
	if err != nil {
		return err
	}
	tb.natsConn = nc

	log.Info("upbound started")

	return nil
}

func (tb *tapirBridge) startDownbound() error {
	nc, err := nats.Connect(tb.natsUrl)
	if err != nil {
		panic(err)
	}

	msgCh := make(chan *nats.Msg)
	_, err = nc.ChanQueueSubscribe(tb.subject, tb.queue, msgCh)
	if err != nil {
		panic(err)
	}

	tb.natsMsgCh = msgCh
	tb.natsConn = nc

	go tb.loopDownbound()

	return nil
}

func (tb *tapirBridge) loopDownbound() {
    if tb.publish == nil {
        panic(errors.New("No publish method set"))
    }

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

			err = tb.publish(tb.topic, payload)
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

		if tb.isUpbound() && filename != "" {
			filenameNodeman := filename + "_nodeman"
			f, err := os.OpenFile(filenameNodeman, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return errors.New("Error opening Nodeman keylogfile for writing")
			}
			tb.keylogfileNodeman = f
		}

		tb.enableKeylogfile = enable

		log.Warning("Logging TLS keys")
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
		if tb.isUpbound() {
			if filename == "" {
				return nil
			}
		}

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

func NodemanApiUrl(urlRaw string) bridgeOpt {
	fptr := func(tb *tapirBridge) error {
		if !tb.isUpbound() {
			if urlRaw != "" {
                log.Info("Nodeman URL '%s' will be ignored for downbound bridges", urlRaw)
			}
		}

		u, err := url.Parse(urlRaw)
		if err != nil {
			return errors.New("Invalid Nodeman API URL")
		}

		tlsCfg := tls.Config{
			MinVersion: tls.VersionTLS13,
		}

		if tb.enableKeylogfile {
			tlsCfg.KeyLogWriter = tb.keylogfileNodeman
		}

		tr := &http.Transport{
			MaxIdleConns:       10,
			DisableCompression: true,
			TLSClientConfig:    &tlsCfg,
		}

		client := http.Client{
			Transport: tr,
		}

		tb.nodemanApi = u
		tb.httpClient = client

		log.Info("Nodeman Api URL configured: '%s'", u)
		return nil
	}

	return fptr
}
