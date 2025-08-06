package itests

import (
    "os"
    "path/filepath"
	"testing"

	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/inject/mqtt"
	"github.com/dnstapir/mqtt-bridge/inject/nats"
	"github.com/dnstapir/mqtt-bridge/inject/fake"
	"github.com/dnstapir/mqtt-bridge/shared"
)


type iTest struct {
    *testing.T
    mqttClient shared.MqttIF
    natsClient shared.NatsIF
    workdir string
    natsTargetDir string
    valkey keys.ValKey
    signkey keys.SignKey
}

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

const c_DIR_BASE = "sut/"
const c_DIR_MOSQUITTO = "mosquitto/"
const c_DIR_NATS = "nats/"
const c_DIR_MQTT_BRIDGE = "mqtt-bridge/"
const c_FILE_TESTKEY = "testkey.json"

// TODO preparation
//// 1. Create temporary folder for test (TMPDIR)
//// 2. Copy contents of mosquitto+nats+mqtt-bridge folders to TMPDIR
//// 3. Generate key and put in TMPDIR/mqtt-bridge
// 4. Start nats and mosquitto containers, mount respective dirs from TMPDIR
// 5. Start generic container for running binary, mount dir from TMPDIR
// 7. Place test binary from OUT in container (e.g. in /usr/bin)
// 8. RUN TESTS, use key from TMPDIR/mqtt-bridge to validate
func (t *iTest) setup() {
    keys.SetLogger(fake.Logger())
    t.setupClients()
    t.setupWorkdir()
    t.setupKeys()
}

func (t *iTest) setupClients() {
	mqttConf := mqtt.Conf{
		Log:            fake.Logger(),
        MqttUrl:        "mqtt://localhost:1883",
	}
    mqttClient, err := mqtt.Create(mqttConf)
	if err != nil {
        panic(err)
	}
    err = mqttClient.Connect()
	if err != nil {
        panic(err)
	}
    t.mqttClient = mqttClient

	natsConf := nats.Conf{
		Log:     fake.Logger(),
        NatsUrl: "nats://localhost:4222",
	}
    natsClient, err := nats.Create(natsConf)
	if err != nil {
        panic(err)
	}
    err = natsClient.Connect()
	if err != nil {
        panic(err)
	}
    t.natsClient = natsClient
}

func (t *iTest) setupWorkdir() {
    t.workdir = t.TempDir()

    for _, dir := range []string{c_DIR_NATS, c_DIR_MOSQUITTO, c_DIR_MQTT_BRIDGE} {
        targetDir := filepath.Join(t.workdir, dir)

        execDir, err := os.Getwd()
	    if err != nil {
            panic(err)
	    }
        sourceDir := filepath.Join(execDir, c_DIR_BASE, dir)

        err = os.CopyFS(targetDir, os.DirFS(sourceDir))
	    if err != nil {
            panic(err)
	    }
    }
}

func (t *iTest) setupKeys() {
    key, err := keys.GenerateSignKey(filepath.Join(t.workdir, c_DIR_MQTT_BRIDGE, c_FILE_TESTKEY))
	if err != nil {
        panic(err)
    }

    t.signkey = key
    t.valkey = keys.ValKey(key)
}

func (t *iTest) setupContainers() {
}

func (t *iTest) teardown() {
    t.mqttClient = nil // TODO call Stop() on client or something?
    t.natsClient = nil // TODO call Stop() on client or something?
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
    it := new(iTest)
    it.T = t /* upgrade to our custom test class */
    it.setup()
    defer it.teardown()

//    inCh, err := natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
//    if err != nil {
//        panic(err)
//    }
//
    outCh, err := it.mqttClient.Subscribe("observations/down/tapir-pop")
    if err != nil {
        panic(err)
    }

    //inCh <- []byte(msgTmpl)

    wanted := []byte{0x65}
    got := <-outCh

    if !compareBytes(got, wanted) {
        t.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(got))
    }
}
