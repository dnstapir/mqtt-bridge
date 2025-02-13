# An example deployment of how to bridge from NATS->MQTT

./bin/mqtt-bridge -mqtt-url "tls://mqtt.dev.dnstapir.se:8883" \
    -debug \
    -direction down
    -data-schema misc/accept-all-schema.json
    -data-key certs/sign.jwk \
    -nats-subject observations.down.tapir-pop \
    -mqtt-topic "observations/down/tapir-pop"
