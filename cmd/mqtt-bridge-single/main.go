package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dnstapir/mqtt-bridge/app"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	var application app.App
	var br app.Bridge

	flag.BoolVar(&application.Debug,
		"debug",
		false,
		"Enable debug logs",
	)
	flag.StringVar(&application.MqttUrl,
		"mqtt-url",
		"mqtts://127.0.0.1:8883",
		"URL of MQTT broker",
	)
	flag.StringVar(&application.MqttCaCert,
		"mqtt-ca-cert",
		"certs/ca.crt",
		"Server mTLS cert for MQTT broker",
	)
	flag.StringVar(&application.MqttClientCert,
		"mqtt-client-cert",
		"certs/tls.crt",
		"Client mTLS cert for MQTT broker",
	)
	flag.StringVar(&application.MqttClientKey,
		"mqtt-client-key",
		"certs/tls.key",
		"Client mTLS key for MQTT broker",
	)
	flag.BoolVar(&application.MqttEnableTlsKlf,
		"enable-keylog-file-yes-i-know-not-for-production",
		false,
		"Enable keylogfile for MQTT",
	)
	flag.StringVar(&application.NatsUrl,
		"nats-url",
		"nats://127.0.0.1:4222",
		"URL of NATS server",
	)

	flag.StringVar(&br.Direction,
		"direction",
		"down",
		"'down' (NATS->MQTT) or 'up' (MQTT->NATS)",
	)
	flag.StringVar(&br.MqttTopic,
		"mqtt-topic",
		"observations/down/tapir-pop",
		"MQTT topic to publish to",
	)
	flag.StringVar(&br.NatsSubject,
		"nats-subject",
		"observations.down.tapir-pop",
		"NATS subject string",
	)
	flag.StringVar(&br.NatsQueue,
		"nats-queue",
		"observations-queue",
		"NATS durable consumer identifier",
	)
	flag.StringVar(&br.Key,
		"data-key",
		"",
		"Key for signing/validating data sent via MQTT",
	)
	flag.StringVar(&br.Schema,
		"data-schema",
		"",
		"Schema for checking data",
	)
	flag.StringVar(&application.NodemanApiUrl,
		"nodeman-api-url",
		"",
		"Endpoint for Nodeman API",
	)

	flag.Parse()

	klfPath := os.Getenv("TAPIR_MQTT_BRIDGE_KLF")
	if klfPath != "" {
		application.MqttTlsKlfPath = klfPath
	}

	application.Bridges = []app.Bridge{br}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application.Ctx = ctx

	fmt.Println("###### starting mqtt-bridge...")
	go application.Run()

	s := <-c
	fmt.Println(fmt.Sprintf("###### mqtt-bridge got signal '%s', exiting...", s))
}
