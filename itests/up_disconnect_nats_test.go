// +build itests

package itests

import (
    "bytes"
    "testing"

	"github.com/dnstapir/mqtt-bridge/app/keys"
)

func TestIntegrationUpBasicWithoutSchemaDisconnectNats(t *testing.T) {
    t.Skip()
    it := new(iTest)
    it.tester = t /* upgrade to our custom test class */
    it.setup(true)
    defer it.teardown()

    inChMqtt, err := it.mqttClient.StartPublishing("events/up/" + it.signkey.KeyID(), false)
    if err != nil {
        panic(err)
    }

    outChNats, err := it.natsClient.Subscribe("events.up.some_event", "eventQ")
    if err != nil {
        panic(err)
    }

    indata := []byte("{\"lala\": 1}")
    signedIndata, err := keys.Sign(indata, it.signkey)
    if err != nil {
        panic(err)
    }

    go func(){inChMqtt <- signedIndata}()

    got := <-outChNats

    wanted := indata
    if !bytes.Equal(wanted, got) {
        it.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(got))
    }

    it.restartService("nats")

    go func(){inChMqtt <- signedIndata}()

    got = <-outChNats

    wanted = indata
    if !bytes.Equal(wanted, got) {
        it.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(got))
    }
}
