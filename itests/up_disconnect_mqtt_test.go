// +build itests

package itests

import (
    "bytes"
    "testing"
	"github.com/dnstapir/mqtt-bridge/app/keys"
)

func TestIntegrationUpBasicWithoutSchemaDisconnectMqtt(t *testing.T) {
    it := new(iTest)
    it.tester = t /* upgrade to our custom test class */
    it.setup(true)
    defer it.teardown()

    inChMqtt, err := it.mqttClient.StartPublishing("events/up/" + it.signkey.KeyID())
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

    it.restartService("mosquitto")

    go func(){inChMqtt <- signedIndata}()

    it.Logf("Waiting for response data...")

    got = <-outChNats

    it.Logf("Got it!")

    wanted = indata
    if !bytes.Equal(wanted, got) {
        it.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(got))
    }
}
