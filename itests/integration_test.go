package itests

import (
	"testing"

	"github.com/dnstapir/mqtt-bridge/inject/mqtt"
	"github.com/dnstapir/mqtt-bridge/inject/nats"
	"github.com/dnstapir/mqtt-bridge/inject/fake"
	"github.com/dnstapir/mqtt-bridge/shared"
)

var mqttClient shared.MqttIF
var natsClient shared.NatsIF

const msgTmpl string = `
{
    "src_name": "dns-tapir",
    "creator": "",
    "msg_type": "observation",
    "list_type": "doubtlist",
    "added": [
        {
            "name": "leon.xa",
            "time_added": "2025-08-05T00:06:21,347168698+02:00",
            "ttl": 3600,
            "tag_mask": 666,
            "extended_tags": []
        }
    ],
    "removed": [],
    "msg": "",
    "timestamp": "2025-08-05T00:06:21,347168698+02:00",
    "time_str": ""
}`

// TODO preparation
// 1. Create temporary folder for test (TMPDIR)
// 2. Copy contents of mosquitto+nats+mqtt-bridge folders to TMPDIR
// 3. Generate key and put in TMPDIR/mqtt-bridge
// 4. Start nats and mosquitto containers, mount respective dirs from TMPDIR
// 5. Start generic container for running binary, mount dir from TMPDIR
// 7. Place test binary from OUT in container (e.g. in /usr/bin)
// 8. RUN TESTS, use key from TMPDIR/mqtt-bridge to validate
func setup() {
    var err error
	mqttConf := mqtt.Conf{
		Log:            fake.Logger(),
        MqttUrl:        "mqtt://localhost:1883",
	}
	mqttClient, err = mqtt.Create(mqttConf)
	if err != nil {
        panic(err)
	}

    err = mqttClient.Connect()
	if err != nil {
        panic(err)
	}

	natsConf := nats.Conf{
		Log:     fake.Logger(),
        NatsUrl: "nats://localhost:4222",
	}
	natsClient, err = nats.Create(natsConf)
	if err != nil {
        panic(err)
	}

    err = natsClient.Connect()
	if err != nil {
        panic(err)
	}
}

func teardown() {
    mqttClient = nil // TODO call Stop() on client or something?
    natsClient = nil // TODO call Stop() on client or something?
}

func compareBytes(a, b []byte) bool {
    if a == nil && b == nil {
        return true
    }

    if len(a) != len(b) {
        return false
    }

    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }

    return true
}

func TestIntegrationDownBasic(t *testing.T) {
    setup()
    defer teardown()

    inCh, err := natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
    if err != nil {
        panic(err)
    }

    outCh, err := mqttClient.Subscribe("observations/down/tapir-pop")
    if err != nil {
        panic(err)
    }

    inCh <- []byte(msgTmpl)

    wanted := []byte{0x65}
    got := <-outCh

    if !compareBytes(got, wanted) {
        t.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(got))
    }
}
