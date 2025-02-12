# An example deployment of how to bridge from MQTT->NATS

mqtt-sender -mqtt-url "tls://mqtt.dev.dnstapir.se:8883" \
    -debug \
    -direction up \
    -data-key certs/eldorado-jet.pub.json \
    -data-schema misc/accept-all-schema.json \
    -nats-subject foo \
    -mqtt-topic "events/up/eldorado-jet.edge.dev.dnstapir.se/leon"
