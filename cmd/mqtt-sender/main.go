package main


import (
    "flag"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "tapir-core-mqtt-sender/app"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

    var app app.App

    flag.BoolVar(&app.Debug,
        "debug",
        false,
        "Enable debug logs",
    )
    flag.StringVar(&app.MqttUrl,
        "mqtt-url",
        "mqtts://127.0.0.1:8883",
        "URL of MQTT broker",
    )
    flag.StringVar(&app.MqttTopic,
        "mqtt-topic",
        "observations/down/tapir-pop",
        "MQTT topic to publish to",
    )
    flag.StringVar(&app.MqttCaCert,
        "mqtt-ca-cert",
        "certs/ca.crt",
        "Server mTLS cert for MQTT broker",
    )
    flag.StringVar(&app.MqttClientCert,
        "mqtt-client-cert",
        "certs/tls.crt",
        "Client mTLS cert for MQTT broker",
    )
    flag.StringVar(&app.MqttClientKey,
        "mqtt-client-key",
        "certs/tls.key",
        "Client mTLS key for MQTT broker",
    )
    flag.BoolVar(&app.MqttEnableTlsKlf,
        "enable-keylog-file-yes-i-know-not-for-production",
        false,
        "Enable keylogfile for MQTT",
    )
    flag.StringVar(&app.MqttSigningKey,
        "mqtt-signing-key",
        "certs/sign.jwk",
        "Key for signing data sent via MQTT",
    )
    flag.StringVar(&app.NatsUrl,
        "nats-url",
        "nats://127.0.0.1:4222",
        "URL of NATS server",
    )
    flag.StringVar(&app.NatsBucket,
        "nats-bucket",
        "observations",
        "Name of NATS bucket to track",
    )
    flag.StringVar(&app.NatsSubject,
        "nats-subject",
        "tapir.core.events.observations.>",
        "NATS subject string",
    )

    flag.Parse()

    klfPath := os.Getenv("TAPIR_MQTT_SENDER_KLF")
    if klfPath != "" {
        app.MqttTlsKlfPath = klfPath
    }

    fmt.Println("###### starting mqtt-sender...")
    app.Run()

    s := <-c
    fmt.Println(fmt.Sprintf("###### mqtt-sender got signal '%s', exiting...", s))
}
