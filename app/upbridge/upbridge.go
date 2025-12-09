package upbridge

import (
	"errors"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/dnstapir/mqtt-bridge/app/cache"
	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/app/schemaval"
)

type upbridge struct {
	log       shared.LoggerIF
	stopCh    chan bool
	key       keys.ValKey
	schemaval *schemaval.Schemaval
	lru       *cache.LruCache
	nodeman   shared.NodemanIF
}

type Conf struct {
	Log     shared.LoggerIF
	Nodeman shared.NodemanIF
	Schema  string
	Key     string
}

func Create(conf Conf) (*upbridge, error) {
	newUpbridge := new(upbridge)

	if conf.Log == nil {
		return nil, errors.New("error setting logger")
	}
	newUpbridge.log = conf.Log

	if conf.Nodeman == nil {
		return nil, errors.New("error setting nodeman handle")
	}
	newUpbridge.nodeman = conf.Nodeman

	newUpbridge.stopCh = make(chan bool, 1)

	cacheConf := cache.Conf{}
	lruCache, err := cache.Create(cacheConf)
	if err != nil {
		return nil, errors.New("error creating key cache")
	}
	newUpbridge.lru = lruCache

	if conf.Key != "" {
		key, err := keys.GetValKey(conf.Key)
		if err != nil {
			return nil, errors.New("error getting validation key")
		}

		err = newUpbridge.lru.StoreValkeyInCache(key)
		if err != nil {
			return nil, errors.New("error storing validation key")
		}
	}

	schemaConf := schemaval.Conf{
		Log:      conf.Log,
		Filename: conf.Schema,
	}
	schema, err := schemaval.Create(schemaConf)
	if err != nil {
		return nil, err
	}
	newUpbridge.schemaval = schema

	return newUpbridge, nil
}

func (ub *upbridge) Start(mqttCh <-chan shared.MqttData, natsCh chan<- shared.NatsData) {
	for {
		select {
		case <-ub.stopCh:
			ub.log.Info("Stopping upbound bridge")
			return
		case mqttData := <-mqttCh:
            outgoingMsg := shared.NatsData {
                Payload: nil,
                Headers: make(map[string]string),
            }

            sig := mqttData.Payload
			keyID, err := keys.GetKeyIDFromSignedData(sig)
			ub.log.Debug("Got MQTT message from '%s'", keyID)
			if err != nil {
				ub.log.Error("Error getting key ID from signed data, err: '%s'", err)
				continue
			}

			key := ub.lru.GetValkeyFromCache(keyID)
			if key == nil {
				ub.log.Info("Key not found in cache, contacting nodeman", keyID)
				newKeyBytes, err := ub.nodeman.GetKey(keyID)
				if err != nil {
					ub.log.Error("Error getting key '%s' from Nodeman, err: %s", keyID, err)
					continue
				}

				newKey, err := keys.ParseValKey(newKeyBytes)
				if err != nil {
					ub.log.Error("Error parsing key '%s', err: %s", keyID, err)
					continue
				}

				err = ub.lru.StoreValkeyInCache(newKey)
				if err != nil {
					ub.log.Error("Error caching key '%s', err: %s", keyID, err)
					continue
				}

				key = newKey
			}

			data, err := keys.CheckSignature(sig, key)
			if err != nil {
				ub.log.Error("Bad signature from MQTT, err: '%s'", err)
				continue
			}
			ub.log.Debug("Signature with ID '%s' ok", keyID)

            outgoingMsg.Headers[shared.NATSHEADER_DNSTAPIR_MESSAGE_SCHEMA] = ub.schemaval.GetID()
            outgoingMsg.Headers[shared.NATSHEADER_DNSTAPIR_MQTT_TOPIC] = mqttData.Topic
            outgoingMsg.Headers[shared.NATSHEADER_DNSTAPIR_KEY_IDENTIFIER] = keyID
            outgoingMsg.Headers[shared.NATSHEADER_DNSTAPIR_KEY_THUMBPRINT] = keys.GetThumbprint(key)

			ok := ub.schemaval.Validate(data)
			if ok {
                outgoingMsg.Payload = data
				natsCh <- outgoingMsg
				ub.log.Debug("Handed over %d bytes to NATS", len(data))
			} else {
				ub.log.Error("Malformed data from MQTT, discarding...")
			}
			ub.log.Debug("Processing of message from '%s' done!", keyID)
		}
	}
}

func (ub *upbridge) Stop() {
	ub.stopCh <- true
	close(ub.stopCh)
}
