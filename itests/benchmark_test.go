// +build itests

package itests

import (
    "fmt"
    "sync"
	"testing"

	"github.com/dnstapir/mqtt-bridge/app/keys"
    "github.com/dnstapir/mqtt-bridge/shared"
)

func BenchmarkDownBasic(b *testing.B) {
    it := new(iTest)
    it.tester = b /* upgrade to our custom test class */
    it.bencher = b /* upgrade to our custom test class */
    it.setup(false)
    n_MESSAGES := 50000
    defer it.teardown()

    inCh, err := it.natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
    if err != nil {
        panic(err)
    }

    outCh, err := it.mqttClient.Subscribe("observations/down/tapir-pop")
    if err != nil {
        panic(err)
    }

    it.ResetTimer()

    var count = 0
	var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for range outCh {
            count++
            if count == n_MESSAGES {
                it.Logf("All messages received!")
                break
            }
        }
    }()

    for i := range n_MESSAGES {
        msg := shared.NatsData {
            Payload: []byte(fmt.Sprintf(msgTmpl, i)),
        }
        inCh <- msg
    }

    wg.Wait()

    it.mqttClient.Stop()
    close(inCh)

    /* For sanity */
    if n_MESSAGES != count {
        it.Fatalf("Sent %d messages but only received %d", n_MESSAGES, count)
    }
}

func BenchmarkUpBasicWithoutSchema(b *testing.B) {
    it := new(iTest)
    it.tester = b /* upgrade to our custom test class */
    it.bencher = b /* upgrade to our custom test class */
    it.setup(false)
    n_MESSAGES := 50000
    defer it.teardown()

    preparedData := make([][]byte, n_MESSAGES)
    for i := range n_MESSAGES {
        indata := []byte(fmt.Sprintf("{\"lala\": %d}", i))
        signedIndata, err := keys.Sign(indata, it.signkey)
        if err != nil {
            panic(err)
        }

        preparedData[i] = signedIndata
    }

    inChMqtt, err := it.mqttClient.StartPublishing("events/up/" + it.signkey.KeyID())
    if err != nil {
        panic(err)
    }

    outChNats, err := it.natsClient.Subscribe("events.up.some_event", "eventQ")
    if err != nil {
        panic(err)
    }

    it.ResetTimer()

    var count = 0
	var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for range outChNats {
            count++
            if count == n_MESSAGES {
                it.Logf("All messages received!")
                break
            }
        }
    }()

    for _, d := range preparedData {
        inChMqtt <- d
    }

    wg.Wait()

    it.natsClient.Stop()
    close(inChMqtt)

    /* For sanity */
    if n_MESSAGES != count {
        it.Fatalf("Sent %d messages but only received %d", n_MESSAGES, count)
    }
}
