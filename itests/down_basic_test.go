// +build itests

package itests

import (
    "bytes"
    "testing"

	"github.com/dnstapir/mqtt-bridge/app/keys"
    "github.com/dnstapir/mqtt-bridge/shared"
)

func TestIntegrationDownBasic(t *testing.T) {
    it := new(iTest)
    it.tester = t /* upgrade to our custom test class */
    it.setup(true)
    defer it.teardown()

    inCh, err := it.natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
    if err != nil {
        panic(err)
    }

    outCh, err := it.mqttClient.Subscribe("observations/down/tapir-pop")
    if err != nil {
        panic(err)
    }

    msg := shared.NatsData {
        Payload: []byte(msgTmpl),
    }
    inCh <- msg

    wanted := msg.Payload
    got := <-outCh

    data, err := keys.CheckSignature(got.Payload, it.valkey)
    if err != nil {
        panic(err)
    }

    if !bytes.Equal(data, wanted) {
        t.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(data))
    }
}
