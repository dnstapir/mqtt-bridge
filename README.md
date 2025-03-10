# Example usage
Coming soon...

# Sample config
```toml
# Enable debug output
Debug = true

# Url of the MQTT broker
MqttUrl = "mqtt://localhost:8883"

# Root certificate for the MQTT mTLS PKI
MqttCaCert = "path/to/ca/cert"

# Client certificate for authentication when connecting to the MQTT broker
MqttClientCert = "path/to/client/cert"

# Private key for authentication when connecting to the MQTT broker
MqttClientKey = "path/to/client/key"

# URL of the NATS server
NatsUrl = "nats://localhost:4222"

# URL of the Nodeman API (only used by upbound bridges)
NodemanApiUrl = "https://localhost/api/v1"

# An upbound bridge
[[Bridges]]
# Direction to bridge in, MQTT->NATS (up) or NATS->MQTT (down)
Direction = "up"

# MQTT topic used by bridge (wildcards possible for "up" bridges)
MqttTopic = "events/up/my-id"

# NATS subject used by bridge (wildcards possible for "down" bridges)
NatsSubject = "events.up.some_event"

# NATS queue group for load balancing (only used for "down" bridges)
NatsQueue = ""

# Key to sign (downbound bridges) or validate (upbound bridges) data
# Upbound bridges can also use the Nodeman API to fetch validation keys
Key = "path/to/data/key"

# Schema to validate data against
Schema = "path/to/json/schema"

# Another bridge, but downbound
[[Bridges]]
Direction = "down"
MqttTopic = "observations/down/tapir-pop"
NatsSubject = "observations.down.tapir-pop"
NatsQueue = "observationsQ"
Key = "path/to/data/key"
Schema = "path/to/json/schema"
```
