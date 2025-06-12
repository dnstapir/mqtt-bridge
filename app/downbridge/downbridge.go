package downbridge

import (
    "errors"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/app/schemaval"
)

type downbridge struct {
    log       shared.ILogger
    stopCh    chan bool
    key       keys.SignKey
    schemaval *schemaval.Schemaval
}

type Conf struct {
    Log      shared.ILogger
    Schema   string
    Key      string
}

func Create(conf Conf) (*downbridge, error) {
    newDownbridge := new(downbridge)

    if conf.Log == nil {
        return nil, errors.New("error setting logger")
    }
    newDownbridge.log = conf.Log

    newDownbridge.stopCh = make(chan bool, 1)

    key, err := keys.GetSignKey(conf.Key)
    if err != nil {
        return nil, errors.New("error getting signing key")
    }
    newDownbridge.key = key

    schemaConf:= schemaval.Conf {
        Log: conf.Log,
        Filename: conf.Schema,
    }
    schema, err := schemaval.Create(schemaConf)
    if err != nil {
        return nil, errors.New("error creating schema")
    }
    newDownbridge.schemaval = schema

    return newDownbridge, nil
}

func (db *downbridge) Start(natsCh <-chan []byte, mqttCh chan<- []byte) {
	for {
		select {
		case <-db.stopCh:
			db.log.Info("Stopping downbound bridge")
			return
        case data := <-natsCh:
	        db.log.Debug("Got message '%s'", string(data))

	        ok := db.schemaval.Validate(data)
	        if ok {
                outData, err := keys.Sign(data, db.key)
                if err == nil {
                    mqttCh <- outData
                } else {
                    db.log.Error("Error signing data from NATS, discarding...")
                }
	        } else {
                db.log.Error("Malformed data from NATS, discarding...")
            }
		}
	}

    // TODO also close other channels?
}

func (db *downbridge) Stop() {
    db.stopCh <- true
    close(db.stopCh)
}
