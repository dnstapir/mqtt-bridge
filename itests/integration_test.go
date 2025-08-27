// +build itests

package itests

import (
    "bytes"
    "context"
    "os"
    "path/filepath"
	"testing"
    "time"

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
            "time_added": "2025-08-05T00:06:21.347168698+02:00",
            "ttl": 3600,
            "tag_mask": 666,
            "extended_tags": []
        }
    ],
    "removed": [],
    "msg": "",
    "timestamp": "2025-08-05T00:06:21.347168698+02:00",
    "time_str": ""
}`

const c_DIR_BASE = "sut/"
const c_DIR_MQTT_BRIDGE = "mqtt-bridge/"
const c_DIR_OUT = "../out/"
const c_FILE_COMPOSE = "docker-compose.yaml"
const c_FILE_TESTKEY = "testkey.json"
const c_FILE_TESTKEY_KID = "tmp-key-itest" /* must match upbridge topic in config */
const c_FILE_DOCKER = "Dockerfile"
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

    targetDir := filepath.Join(t.workdir, c_DIR_BASE)

    execDir, err := os.Getwd()
	if err != nil {
        panic(err)
	}
    sourceDir := filepath.Join(execDir, c_DIR_BASE)

    err = os.CopyFS(targetDir, os.DirFS(sourceDir))
	if err != nil {
        panic(err)
	}
}

func (t *iTest) setupKeys() {
    key, err := keys.GenerateSignKey(filepath.Join(t.workdir, c_DIR_BASE, c_DIR_MQTT_BRIDGE, c_FILE_TESTKEY), c_FILE_TESTKEY_KID)
	if err != nil {
        panic(err)
    }

    t.signkey = key
    t.valkey = keys.ValKey(key)
}

func (t *iTest) setupContainers() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    copyFile(filepath.Join(c_DIR_BASE, c_FILE_DOCKER), filepath.Join(c_DIR_OUT, c_FILE_DOCKER))

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

    stack, err := compose.NewDockerCompose(filepath.Join(t.workdir, c_DIR_BASE, c_FILE_COMPOSE))
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

func copyFile(src, dst string) {
    data, err := os.ReadFile(src)
	if err != nil {
        panic(err)
	}

    err = os.WriteFile(dst, data, 0666)
	if err != nil {
        panic(err)
	}
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

    wanted := []byte(msgTmpl)
    got := <-outCh

    data, err := keys.CheckSignature(got, it.valkey)
    if err != nil {
        panic(err)
    }

    if !bytes.Equal(data, wanted) {
        t.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(data))
    }
}

func TestIntegrationUpBasicWithoutSchema(t *testing.T) {
    it := new(iTest)
    it.T = t /* upgrade to our custom test class */
    it.setup()
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

func TestIntegrationDownWithoutMqttConnectionOutage(t *testing.T) {
    it := new(iTest)
    it.T = t /* upgrade to our custom test class */
    it.setup()
    defer it.teardown()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    inCh, err := it.natsClient.StartPublishing("observations.down.tapir-pop", "observationsQ")
    if err != nil {
        panic(err)
    }

    outCh, err := it.mqttClient.Subscribe("observations/down/tapir-pop")
    if err != nil {
        panic(err)
    }

    inCh <- []byte(msgTmpl)

    wanted := []byte(msgTmpl)
    got := <-outCh

    data, err := keys.CheckSignature(got, it.valkey)
    if err != nil {
        panic(err)
    }

    if !bytes.Equal(data, wanted) {
        t.Fatalf("wanted: '%s', got: '%s'", string(wanted), string(data))
    }

    mosquittoContainer, err := it.stack.ServiceContainer(ctx, "mosquitto")
    if err != nil {
        panic(err)
    }

    err = mosquittoContainer.Stop(ctx, nil)
    if err != nil {
        panic(err)
    }

    err = mosquittoContainer.Start(ctx)
    if err != nil {
        panic(err)
    }

    inCh <- []byte(msgTmpl)

    wanted = []byte(msgTmpl)
    got = <-outCh

    data, err = keys.CheckSignature(got, it.valkey)
    if err != nil {
        panic(err)
    }

    if !bytes.Equal(data, wanted) {
        t.Fatalf("After mqtt restart, wanted: '%s', got: '%s'", string(wanted), string(data))
    }
}
