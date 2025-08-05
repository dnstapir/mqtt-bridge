package itests

import (
	"context"
	"testing"
    "net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/nats-io/nats.go"
)

// TODO Maybe mutex protext or something...
var responseBuffer [][]byte

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

func mqttClientWaitResponse() {
    for {
        if len(responseBuffer) != 0 {
            break
        }
    }

    return
}

func mqttClientStart() {
    u, err := url.Parse("mqtt://localhost:1883")
	if err != nil {
		panic(err)
	}


	cliCfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{u},
		KeepAlive:  20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval: 60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: "observations/down/tapir-pop", QoS: 0},
				},
			}); err != nil {
                panic(err)
			}
		},
		OnConnectError: func(err error) { panic(err) },
		ClientConfig: paho.ClientConfig{
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
                    responseBuffer = append(responseBuffer, pr.Packet.Payload)
					return true, nil
				}},
			OnClientError: func(err error) { panic(err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
                panic("disconnect")
			},
		},
	}

	c, err := autopaho.NewConnection(context.Background(), cliCfg)
	if err != nil {
		panic(err)
	}
	if err = c.AwaitConnection(context.Background()); err != nil {
		panic(err)
	}
}

func natsClientSend(msg string) {
    nc, _ := nats.Connect("nats://localhost:4222")

	defer nc.Drain()

	nc.Publish("observations.down.tapir-pop", []byte(msg))
}

func TestIntegrationUpBasic(t *testing.T) {
    responseBuffer = make([][]byte, 0)

    mqttClientStart()

    natsClientSend(msgTmpl)

    mqttClientWaitResponse()

    wanted := []byte("lolo")
    if true {
		t.Fatalf("Bad response, wanted '%s', got '%s'", wanted, responseBuffer[0])
    }
}
