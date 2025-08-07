package itests

import (
    "context"
    "os"
    "path/filepath"
	"testing"

	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/inject/mqtt"
	"github.com/dnstapir/mqtt-bridge/inject/nats"
	"github.com/dnstapir/mqtt-bridge/inject/fake"
	"github.com/dnstapir/mqtt-bridge/shared"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/compose"
)


type iTest struct {
    *testing.T
    mqttClient shared.MqttIF
    natsClient shared.NatsIF
    workdir string
    natsTargetDir string
    valkey keys.ValKey
    signkey keys.SignKey
    stack *compose.DockerCompose
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
const c_DIR_OUT = "../out"
const c_FILE_TESTKEY = "testkey.json"
const c_IMAGE_TESTDOCKER_REPO = "mqtt-bridge"
const c_IMAGE_TESTDOCKER_TAG = "itest"

func (t *iTest) setup() {
    keys.SetLogger(fake.Logger())
    t.setupWorkdir()
    t.setupKeys()
    t.setupContainers()
    t.setupClients()
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
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        FromDockerfile: testcontainers.FromDockerfile{
            Context:    filepath.Join(".", c_DIR_OUT),
            Repo:       c_IMAGE_TESTDOCKER_REPO,
            Tag:        c_IMAGE_TESTDOCKER_TAG,
        },
    }
    prov, err := testcontainers.NewDockerProvider()
    if err != nil {
        panic(err)
    }
    _, err = prov.BuildImage(ctx, &req)
    if err != nil {
        panic(err)
    }

    stack, err := compose.NewDockerCompose("sut/docker-compose.yaml")
    if err != nil {
        panic(err)
    }

    err = stack.Up(ctx, compose.Wait(true))
    if err != nil {
        panic(err)
    }

    t.stack = stack
}

func (t *iTest) teardown() {
    t.mqttClient = nil // TODO call Stop() on client or something?
    t.natsClient = nil // TODO call Stop() on client or something?
    err := t.stack.Down(
        context.Background(),
        compose.RemoveOrphans(true),
        compose.RemoveVolumes(true),
        compose.RemoveImagesLocal,
    )
    if err != nil {
        panic(err)
    }
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

    inCh, err := it.natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
    if err != nil {
        panic(err)
    }

    outCh, err := it.mqttClient.Subscribe("observations/down/tapir-pop")
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
