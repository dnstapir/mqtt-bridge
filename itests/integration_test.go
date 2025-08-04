package itests

import (
	"context"
	"testing"
    "net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

// TODO Maybe mutex protext or something...
var responseBuffer [][]byte

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

func TestIntegrationUpBasic(t *testing.T) {
    responseBuffer = make([][]byte, 0)

    mqttClientStart()

    //natsClientSend("{\"lala\": 1}")

    mqttClientWaitResponse()

    wanted := []byte("lala")
    if true {
		t.Fatalf("Bad response, wanted '%s', got '%s'", wanted, responseBuffer[0])
    }
}
