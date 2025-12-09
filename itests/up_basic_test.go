// +build itests

package itests

import (
    "bytes"
    "testing"

	"github.com/dnstapir/mqtt-bridge/app/keys"
)

func TestIntegrationUpBasicWithoutSchema(t *testing.T) {
    it := new(iTest)
    it.tester = t /* upgrade to our custom test class */
    it.setup(true)
    defer it.teardown()

    inCh, err := it.mqttClient.StartPublishing("events/up/" + it.signkey.KeyID())
    if err != nil {
        panic(err)
    }

    outCh, err := it.natsClient.Subscribe("events.up.some_event", "eventQ")
    if err != nil {
        panic(err)
    }

    indata := []byte("{\"lala\": 1}")
    signedIndata, err := keys.Sign(indata, it.signkey)
    if err != nil {
        panic(err)
    }

    inCh <- signedIndata

    got := <-outCh

    wanted := indata
    if !bytes.Equal(wanted, got) {
        t.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(got))
    }
}
