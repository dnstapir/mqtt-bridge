// +build itests

package itests

import (
    "bytes"
    "testing"
	"github.com/dnstapir/mqtt-bridge/app/keys"
)

func TestIntegrationDownDisconnect(t *testing.T) {
    it := new(iTest)
    it.tester = t /* upgrade to our custom test class */
    it.setup(true)
    defer it.teardown()

    inChNats, err := it.natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
    if err != nil {
        panic(err)
    }

    outChMqtt, err := it.mqttClient.Subscribe("observations/down/tapir-pop")
    if err != nil {
        panic(err)
    }

    inChNats <- []byte(msgTmpl)

    wanted := []byte(msgTmpl)
    got := <-outChMqtt

    data, err := keys.CheckSignature(got, it.valkey)
    if err != nil {
        panic(err)
    }

    if !bytes.Equal(data, wanted) {
        it.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(data))
    }

    it.restartService("mosquitto")

    inChNats <- []byte(msgTmpl)

    wanted = []byte(msgTmpl)
    got = <-outChMqtt

    data, err = keys.CheckSignature(got, it.valkey)
    if err != nil {
        panic(err)
    }

    if !bytes.Equal(data, wanted) {
        it.Fatalf("After mqtt restart, wanted: '%s', got: '%s'", string(wanted), string(data))
    }
}
