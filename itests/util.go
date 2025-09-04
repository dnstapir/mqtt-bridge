// +build itests

package itests

import (
    "context"
    "os"
    "path/filepath"
    "time"

	"github.com/dnstapir/mqtt-bridge/inject/mqtt"
	"github.com/dnstapir/mqtt-bridge/inject/nats"
	"github.com/dnstapir/mqtt-bridge/inject/logging"
	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/shared"
    "github.com/testcontainers/testcontainers-go/modules/compose"
)

type tester interface {
    Logf(format string, args ...any)
    Fatalf(format string, args ...any)
    TempDir() string
}

type bencher interface {
    ResetTimer()
}

type iTest struct {
    tester
    bencher
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
            "name": "leon%d.xa",
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
const c_MAX_MQTT_BRIDGE_CONNECTION_CHECKS = 5

func (t *iTest) setup(debug bool) {
	log := logging.Create(debug, false)

    keys.SetLogger(log)
    t.setupWorkdir()
    t.setupKeys()
    t.setupContainers()
    t.setupClients(log)
}

func (t *iTest) setupClients(log shared.LoggerIF) {
	mqttConf := mqtt.Conf{
		Log:            log,
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
		Log:     log,
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
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    stack, err := compose.NewDockerCompose(filepath.Join(t.workdir, c_DIR_BASE, c_FILE_COMPOSE))
    if err != nil {
        panic(err)
    }

    err = stack.Up(ctx,
                   compose.Wait(true),
                   compose.WithRecreate("nats"),
                   compose.WithRecreate("mosquitto"),
                   compose.WithRecreate("mqtt-bridge"))
    if err != nil {
        panic(err)
    }

    t.stack = stack
}

func (t *iTest) teardown() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    t.mqttClient = nil // TODO call Stop() on client or something?
    t.natsClient = nil // TODO call Stop() on client or something?
    err := t.stack.Down(
        ctx,
        compose.RemoveOrphans(true),
        compose.RemoveVolumes(true),
    )

    if err != nil {
        panic(err)
    }
}

func (t *iTest) restartService(service string) {
    t.Logf("Restarting service '%s'", service)

    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()

    container, err := t.stack.ServiceContainer(ctx, service)
    if err != nil {
        panic(err)
    }

    err = container.Stop(ctx, nil)
    if err != nil {
        panic(err)
    }

    err = container.Start(ctx)
    if err != nil {
        panic(err)
    }

    for i := range c_MAX_MQTT_BRIDGE_CONNECTION_CHECKS {
        if t.mqttClient.CheckConnection() {
            break
        }

        if i == c_MAX_MQTT_BRIDGE_CONNECTION_CHECKS-1 {
            panic("max mqtt connection retries reached after restarting service")
        }
        time.Sleep(5*time.Second)
    }

    t.Logf("Done restarting '%s'!", service)
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
