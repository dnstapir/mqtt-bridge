package upbridge

import (
    "errors"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/app/schemaval"
	"github.com/dnstapir/mqtt-bridge/app/cache"
)

type upbridge struct {
    log       shared.ILogger
    stopCh    chan bool
    key       keys.ValKey
    schemaval *schemaval.Schemaval
    lru       *cache.LruCache
}

type Conf struct {
    Log      shared.ILogger
    Schema   string
    Key      string
}

func Create(conf Conf) (*upbridge, error) {
    newUpbridge := new(upbridge)

    if conf.Log == nil {
        return nil, errors.New("error setting logger")
    }
    newUpbridge.log = conf.Log

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
            return nil, errors.New("error getting signing key")
        }
    }

    schemaConf:= schemaval.Conf {
        Log: conf.Log,
        Filename: conf.Schema,
    }
    schema, err := schemaval.Create(schemaConf)
    if err != nil {
        return nil, errors.New("error creating schema")
    }
    newUpbridge.schemaval = schema

    return newUpbridge, nil
}

func (ub *upbridge) Start(mqttCh <-chan []byte, natsCh chan<- []byte) {
	for {
		select {
		case <-ub.stopCh:
			ub.log.Info("Stopping downbound bridge")
			return
        case sig := <-mqttCh:
            keyID, err := keys.GetKeyIDFromSignedData(sig)
            if err != nil {
                ub.log.Error("Error getting key ID from signed data, err: '%s'", err)
                continue
            }

            key := ub.lru.GetValkeyFromCache(keyID)
            if key == nil {
                ub.log.Error("key '%s not found in cache", keyID)
                continue // TODO get it with nodeman instead
            }

            data, err := keys.CheckSignature(sig, key)
            if err != nil {
                ub.log.Error("Bad signature from MQTT, err: '%s'", err)
                continue
            }

	        ok := ub.schemaval.Validate(data)
	        if ok {
                // TODO set data headers
                natsCh <- data
	        } else {
                ub.log.Error("Malformed data from MQTT, discarding...")
            }
		}
	}

    // TODO also close other channels?
}

func (ub *upbridge) Stop() {
    ub.stopCh <- true
    close(ub.stopCh)
}
